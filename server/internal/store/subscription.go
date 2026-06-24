package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Subscription mirrors a remote share scope (someone else's published schedule)
// into this plan. The token is never exposed via JSON.
type Subscription struct {
	ID              string  `json:"id"`
	SourceLabel     string  `json:"sourceLabel"`
	RemoteURL       string  `json:"remoteUrl"`
	IntervalSeconds int     `json:"intervalSeconds"`
	Paused          bool    `json:"paused"`
	LastSyncedAt    *string `json:"lastSyncedAt"`
	LastStatus      string  `json:"lastStatus"`
	CreatedAt       string  `json:"createdAt"`
}

func (s *Store) CreateSubscription(ctx context.Context, id, label, remoteURL, token string, interval int) (Subscription, error) {
	if interval <= 0 {
		interval = 300
	}
	var ca time.Time
	err := s.pool.QueryRow(ctx,
		`INSERT INTO subscription (id, source_label, remote_url, token, interval_seconds)
		 VALUES ($1, $2, $3, $4, $5) RETURNING created_at`,
		id, label, remoteURL, token, interval).Scan(&ca)
	if err != nil {
		return Subscription{}, err
	}
	return Subscription{ID: id, SourceLabel: label, RemoteURL: remoteURL, IntervalSeconds: interval, CreatedAt: ca.Format(time.RFC3339)}, nil
}

const subColumns = `id, source_label, remote_url, interval_seconds, paused, last_synced_at, last_status, created_at`

func scanSubscription(row pgx.Row) (Subscription, error) {
	var sub Subscription
	var last *time.Time
	var ca time.Time
	if err := row.Scan(&sub.ID, &sub.SourceLabel, &sub.RemoteURL, &sub.IntervalSeconds, &sub.Paused, &last, &sub.LastStatus, &ca); err != nil {
		return sub, err
	}
	sub.CreatedAt = ca.Format(time.RFC3339)
	if last != nil {
		v := last.Format(time.RFC3339)
		sub.LastSyncedAt = &v
	}
	return sub, nil
}

func (s *Store) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	out := []Subscription{}
	rows, err := s.pool.Query(ctx, `SELECT `+subColumns+` FROM subscription ORDER BY created_at DESC`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return out, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) GetSubscription(ctx context.Context, id string) (Subscription, error) {
	sub, err := scanSubscription(s.pool.QueryRow(ctx, `SELECT `+subColumns+` FROM subscription WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	return sub, err
}

func (s *Store) SetSubscriptionPaused(ctx context.Context, id string, paused bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE subscription SET paused = $2 WHERE id = $1`, id, paused)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSubscription removes the subscription and everything it mirrored.
func (s *Store) DeleteSubscription(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Deleting the external lanes cascades their sub-lanes/items/links.
	if _, err := tx.Exec(ctx, `DELETE FROM swimlane WHERE source_system = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM item WHERE source_system = $1`, id); err != nil {
		return err
	}
	ct, err := tx.Exec(ctx, `DELETE FROM subscription WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// ── sync engine ───────────────────────────────────────────────────────────────

type subWire struct {
	Swimlanes []struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Color    string    `json:"color"`
		SubLanes []SubLane `json:"subLanes"`
	} `json:"swimlanes"`
	Milestones []Item `json:"milestones"`
	Links      []Link `json:"links"`
}

// SyncSubscription polls the remote feed (sending If-None-Match) and mirrors it
// locally. A 304 is a cheap no-op. The outcome is recorded in last_status.
func (s *Store) SyncSubscription(ctx context.Context, id string) error {
	var remoteURL, token, etag string
	err := s.pool.QueryRow(ctx, `SELECT remote_url, token, etag FROM subscription WHERE id = $1`, id).
		Scan(&remoteURL, &token, &etag)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	body, newEtag, changed, ferr := fetchFeed(ctx, remoteURL, token, etag)
	if ferr != nil {
		s.markSync(ctx, id, "", "error: "+ferr.Error())
		return ferr
	}
	if !changed {
		s.markSync(ctx, id, etag, "ok (unchanged)")
		return nil
	}
	var wire subWire
	if err := json.Unmarshal(body, &wire); err != nil {
		s.markSync(ctx, id, etag, "error: invalid feed")
		return err
	}
	if err := s.applyMirror(ctx, id, remoteURL, wire); err != nil {
		s.markSync(ctx, id, etag, "error: "+err.Error())
		return err
	}
	s.markSync(ctx, id, newEtag, "ok")
	return nil
}

func (s *Store) markSync(ctx context.Context, id, etag, status string) {
	if etag != "" {
		_, _ = s.pool.Exec(ctx, `UPDATE subscription SET etag = $2, last_status = $3, last_synced_at = now() WHERE id = $1`, id, etag, status)
		return
	}
	_, _ = s.pool.Exec(ctx, `UPDATE subscription SET last_status = $2, last_synced_at = now() WHERE id = $1`, id, status)
}

// fetchFeed GETs <base>/api/shared with the bearer token and an optional ETag.
// SSRF note: only http/https is allowed; production deployments behind untrusted
// networks should additionally block private address ranges.
func fetchFeed(ctx context.Context, base, token, etag string) (body []byte, newEtag string, changed bool, err error) {
	u, err := url.Parse(strings.TrimRight(base, "/") + "/api/shared")
	if err != nil {
		return nil, "", false, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, "", false, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, etag, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", false, fmt.Errorf("remote returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20)) // 16 MiB cap
	if err != nil {
		return nil, "", false, err
	}
	return b, resp.Header.Get("ETag"), true, nil
}

// applyMirror reconciles the feed into local mirrored entities in one tx.
// External lanes are upserted (so the consumer's local order + hidden flag
// survive a re-sync); their sub-lanes/items/links are fully replaced.
func (s *Store) applyMirror(ctx context.Context, subID, remoteURL string, wire subWire) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	existing := map[string]string{} // remote lane id → local lane id
	rows, err := tx.Query(ctx, `SELECT external_id, id FROM swimlane WHERE source_system = $1`, subID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var ext, lid string
		if err := rows.Scan(&ext, &lid); err != nil {
			rows.Close()
			return err
		}
		existing[ext] = lid
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	laneLocal := map[string]string{}
	seen := map[string]bool{}
	for _, sw := range wire.Swimlanes {
		seen[sw.ID] = true
		color := sw.Color
		if color == "" {
			color = "#0A84FF"
		}
		lid, ok := existing[sw.ID]
		if ok {
			if _, err := tx.Exec(ctx, `UPDATE swimlane SET name = $2, color = $3 WHERE id = $1`, lid, sw.Name, color); err != nil {
				return err
			}
		} else {
			lid = uuid.NewString()
			if _, err := tx.Exec(ctx,
				`INSERT INTO swimlane (id, name, color, sort_order, source_system, external_id)
				 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM swimlane), $4, $5)`,
				lid, sw.Name, color, subID, sw.ID); err != nil {
				return err
			}
		}
		laneLocal[sw.ID] = lid
	}
	for ext, lid := range existing {
		if !seen[ext] {
			if _, err := tx.Exec(ctx, `DELETE FROM swimlane WHERE id = $1`, lid); err != nil {
				return err
			}
		}
	}

	// Replace this subscription's items + the kept lanes' sub-lanes.
	if _, err := tx.Exec(ctx, `DELETE FROM item WHERE source_system = $1`, subID); err != nil {
		return err
	}
	laneIDs := make([]string, 0, len(laneLocal))
	for _, lid := range laneLocal {
		laneIDs = append(laneIDs, lid)
	}
	if len(laneIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM sub_lane WHERE swimlane_id = ANY($1)`, laneIDs); err != nil {
			return err
		}
	}

	subLocal := map[string]string{}
	for _, sw := range wire.Swimlanes {
		lid := laneLocal[sw.ID]
		for _, sub := range sw.SubLanes {
			nsid := uuid.NewString()
			subLocal[sub.ID] = nsid
			if _, err := tx.Exec(ctx,
				`INSERT INTO sub_lane (id, swimlane_id, name, sort_order)
				 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM sub_lane WHERE swimlane_id = $2))`,
				nsid, lid, sub.Name); err != nil {
				return err
			}
		}
	}

	itemLocal := map[string]string{}
	now := time.Now()
	for _, it := range wire.Milestones {
		lid, ok := laneLocal[it.SwimlaneID]
		if !ok {
			continue
		}
		var nsub interface{}
		if it.SubLaneID != nil {
			if v, ok := subLocal[*it.SubLaneID]; ok {
				nsub = v
			}
		}
		defaultsForItem(&it)
		whenV, startV, endV, err := itemDates(it)
		if err != nil {
			return err
		}
		nid := uuid.NewString()
		itemLocal[it.ID] = nid
		if _, err := tx.Exec(ctx,
			`INSERT INTO item (`+itemColumns+`)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			nid, lid, nsub, it.Year, it.Month, it.Title, it.What, it.Why, it.How, it.Who,
			whenV, it.Kind, it.Marker, startV, endV, it.Color,
			subID, it.ID, remoteURL, now); err != nil {
			return err
		}
	}

	for _, l := range wire.Links {
		a, ok1 := itemLocal[l.A]
		b, ok2 := itemLocal[l.B]
		if !ok1 || !ok2 || a == b {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO link (a_item_id, b_item_id)
			 SELECT $1, $2
			 WHERE NOT EXISTS (
			   SELECT 1 FROM link WHERE (a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1)
			 )`,
			a, b); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// SyncDueSubscriptions syncs every active subscription whose interval has elapsed.
// Used by the background poller.
func (s *Store) SyncDueSubscriptions(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT id FROM subscription
		 WHERE paused = false
		   AND (last_synced_at IS NULL OR last_synced_at < now() - make_interval(secs => interval_seconds))`)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		_ = s.SyncSubscription(ctx, id)
	}
}
