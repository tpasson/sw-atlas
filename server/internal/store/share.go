package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidToken is returned when a subscribe token is unknown, revoked or expired.
var ErrInvalidToken = errors.New("invalid or expired share token")

// ShareScope is a named, reusable selection of what may be shared. The selection
// is the union of whole lanes and explicit items, minus explicit excludes.
type ShareScope struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DetailLevel string   `json:"detailLevel"` // "timing" | "full"
	CreatedAt   string   `json:"createdAt"`
	Lanes       []string `json:"lanes"`
	Items       []string `json:"items"`
	Excludes    []string `json:"excludes"`
	TokenCount  int      `json:"tokenCount"`
}

// ShareToken is a subscribe token's metadata. The raw secret is never stored or
// returned after creation — only its hash lives in the database.
type ShareToken struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	CreatedAt      string  `json:"createdAt"`
	ExpiresAt      *string `json:"expiresAt"`
	LastAccessedAt *string `json:"lastAccessedAt"`
	Revoked        bool    `json:"revoked"`
}

// ── Scopes ──────────────────────────────────────────────────────────────────

func (s *Store) CreateShareScope(ctx context.Context, sc ShareScope) (ShareScope, error) {
	if sc.DetailLevel != "full" {
		sc.DetailLevel = "timing"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sc, err
	}
	defer tx.Rollback(ctx)

	var ca time.Time
	if err := tx.QueryRow(ctx,
		`INSERT INTO share_scope (id, name, detail_level) VALUES ($1, $2, $3) RETURNING created_at`,
		sc.ID, sc.Name, sc.DetailLevel).Scan(&ca); err != nil {
		return sc, err
	}
	if err := insertScopeRefs(ctx, tx, `share_scope_lane`, `swimlane_id`, sc.ID, sc.Lanes); err != nil {
		return sc, err
	}
	if err := insertScopeRefs(ctx, tx, `share_scope_item`, `item_id`, sc.ID, sc.Items); err != nil {
		return sc, err
	}
	if err := insertScopeRefs(ctx, tx, `share_scope_exclude`, `item_id`, sc.ID, sc.Excludes); err != nil {
		return sc, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sc, err
	}
	sc.CreatedAt = ca.Format(time.RFC3339)
	return sc, nil
}

func insertScopeRefs(ctx context.Context, tx pgx.Tx, table, col, scopeID string, ids []string) error {
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+table+` (scope_id, `+col+`) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			scopeID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListShareScopes(ctx context.Context) ([]ShareScope, error) {
	out := []ShareScope{}
	rows, err := s.pool.Query(ctx,
		`SELECT sc.id, sc.name, sc.detail_level, sc.created_at,
		        COUNT(t.id) FILTER (WHERE t.revoked = false)
		 FROM share_scope sc
		 LEFT JOIN share_token t ON t.scope_id = sc.id
		 GROUP BY sc.id, sc.name, sc.detail_level, sc.created_at
		 ORDER BY sc.created_at DESC`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc ShareScope
		var ca time.Time
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.DetailLevel, &ca, &sc.TokenCount); err != nil {
			return out, err
		}
		sc.CreatedAt = ca.Format(time.RFC3339)
		out = append(out, sc)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	for i := range out {
		if err := s.loadScopeRefs(ctx, &out[i]); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (s *Store) GetShareScope(ctx context.Context, id string) (ShareScope, error) {
	var sc ShareScope
	var ca time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, detail_level, created_at FROM share_scope WHERE id = $1`, id).
		Scan(&sc.ID, &sc.Name, &sc.DetailLevel, &ca)
	if errors.Is(err, pgx.ErrNoRows) {
		return sc, ErrNotFound
	}
	if err != nil {
		return sc, err
	}
	sc.CreatedAt = ca.Format(time.RFC3339)
	if err := s.loadScopeRefs(ctx, &sc); err != nil {
		return sc, err
	}
	return sc, nil
}

func (s *Store) loadScopeRefs(ctx context.Context, sc *ShareScope) error {
	var err error
	if sc.Lanes, err = s.scopeRefs(ctx, `share_scope_lane`, `swimlane_id`, sc.ID); err != nil {
		return err
	}
	if sc.Items, err = s.scopeRefs(ctx, `share_scope_item`, `item_id`, sc.ID); err != nil {
		return err
	}
	sc.Excludes, err = s.scopeRefs(ctx, `share_scope_exclude`, `item_id`, sc.ID)
	return err
}

func (s *Store) scopeRefs(ctx context.Context, table, col, scopeID string) ([]string, error) {
	out := []string{}
	rows, err := s.pool.Query(ctx, `SELECT `+col+` FROM `+table+` WHERE scope_id = $1`, scopeID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) DeleteShareScope(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM share_scope WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Tokens ──────────────────────────────────────────────────────────────────

func (s *Store) CreateShareToken(ctx context.Context, id, scopeID, hash, label string, expiresAt *time.Time) (ShareToken, error) {
	var ca time.Time
	err := s.pool.QueryRow(ctx,
		`INSERT INTO share_token (id, scope_id, token_hash, label, expires_at)
		 VALUES ($1, $2, $3, $4, $5) RETURNING created_at`,
		id, scopeID, hash, label, expiresAt).Scan(&ca)
	if err != nil {
		return ShareToken{}, err
	}
	t := ShareToken{ID: id, Label: label, CreatedAt: ca.Format(time.RFC3339)}
	if expiresAt != nil {
		v := expiresAt.Format(time.RFC3339)
		t.ExpiresAt = &v
	}
	return t, nil
}

func (s *Store) ListShareTokens(ctx context.Context, scopeID string) ([]ShareToken, error) {
	out := []ShareToken{}
	rows, err := s.pool.Query(ctx,
		`SELECT id, label, created_at, expires_at, last_accessed_at, revoked
		 FROM share_token WHERE scope_id = $1 ORDER BY created_at DESC`, scopeID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var t ShareToken
		var ca time.Time
		var exp, last *time.Time
		if err := rows.Scan(&t.ID, &t.Label, &ca, &exp, &last, &t.Revoked); err != nil {
			return out, err
		}
		t.CreatedAt = ca.Format(time.RFC3339)
		if exp != nil {
			v := exp.Format(time.RFC3339)
			t.ExpiresAt = &v
		}
		if last != nil {
			v := last.Format(time.RFC3339)
			t.LastAccessedAt = &v
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) RevokeShareToken(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE share_token SET revoked = true WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveToken validates a token hash and returns its scope id + detail level,
// bumping last_accessed_at. Unknown/revoked/expired tokens yield ErrInvalidToken.
func (s *Store) ResolveToken(ctx context.Context, hash string) (scopeID, detailLevel string, err error) {
	var revoked bool
	var exp *time.Time
	err = s.pool.QueryRow(ctx,
		`SELECT t.scope_id, t.revoked, t.expires_at, sc.detail_level
		 FROM share_token t JOIN share_scope sc ON sc.id = t.scope_id
		 WHERE t.token_hash = $1`, hash).Scan(&scopeID, &revoked, &exp, &detailLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrInvalidToken
	}
	if err != nil {
		return "", "", err
	}
	if revoked || (exp != nil && exp.Before(time.Now())) {
		return "", "", ErrInvalidToken
	}
	_, _ = s.pool.Exec(ctx, `UPDATE share_token SET last_accessed_at = now() WHERE token_hash = $1`, hash)
	return scopeID, detailLevel, nil
}

// ── Scope resolution ──────────────────────────────────────────────────────────

// ResolveScopePlan builds the plan a scope exposes: the included items (lanes ∪
// items − excludes), the swimlanes/sub-lanes those items reference, and only the
// links whose endpoints are both included. With detailLevel "timing", the
// narrative fields are dropped; provenance is always stripped.
func (s *Store) ResolveScopePlan(ctx context.Context, scopeID, detailLevel string) (Plan, error) {
	p := Plan{Swimlanes: []Swimlane{}, Milestones: []Item{}, Links: []Link{}}

	rows, err := s.pool.Query(ctx,
		`SELECT `+itemColumns+` FROM item
		 WHERE (
		     swimlane_id IN (SELECT swimlane_id FROM share_scope_lane WHERE scope_id = $1)
		     OR id IN (SELECT item_id FROM share_scope_item WHERE scope_id = $1)
		 )
		 AND id NOT IN (SELECT item_id FROM share_scope_exclude WHERE scope_id = $1)
		 ORDER BY year, month, title`, scopeID)
	if err != nil {
		return p, err
	}
	included := map[string]bool{}
	swSet := map[string]bool{}
	subSet := map[string]bool{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return p, err
		}
		if detailLevel != "full" {
			it.What, it.Why, it.How, it.Who, it.When = "", "", "", "", nil
		}
		it.SourceSystem, it.ExternalID, it.ExternalURL, it.LastSyncedAt = nil, nil, nil, nil
		included[it.ID] = true
		swSet[it.SwimlaneID] = true
		if it.SubLaneID != nil {
			subSet[*it.SubLaneID] = true
		}
		p.Milestones = append(p.Milestones, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return p, err
	}

	if len(swSet) > 0 {
		idx := map[string]int{}
		swRows, err := s.pool.Query(ctx,
			`SELECT id, name, color FROM swimlane WHERE id = ANY($1) ORDER BY sort_order, name`, keysOf(swSet))
		if err != nil {
			return p, err
		}
		for swRows.Next() {
			var sw Swimlane
			if err := swRows.Scan(&sw.ID, &sw.Name, &sw.Color); err != nil {
				swRows.Close()
				return p, err
			}
			sw.SubLanes = []SubLane{}
			idx[sw.ID] = len(p.Swimlanes)
			p.Swimlanes = append(p.Swimlanes, sw)
		}
		swRows.Close()
		if err := swRows.Err(); err != nil {
			return p, err
		}
		if len(subSet) > 0 {
			subRows, err := s.pool.Query(ctx,
				`SELECT id, swimlane_id, name FROM sub_lane WHERE id = ANY($1) ORDER BY sort_order, name`, keysOf(subSet))
			if err != nil {
				return p, err
			}
			for subRows.Next() {
				var sub SubLane
				var swID string
				if err := subRows.Scan(&sub.ID, &swID, &sub.Name); err != nil {
					subRows.Close()
					return p, err
				}
				if i, ok := idx[swID]; ok {
					p.Swimlanes[i].SubLanes = append(p.Swimlanes[i].SubLanes, sub)
				}
			}
			subRows.Close()
			if err := subRows.Err(); err != nil {
				return p, err
			}
		}
	}

	lkRows, err := s.pool.Query(ctx, `SELECT a_item_id, b_item_id FROM link`)
	if err != nil {
		return p, err
	}
	for lkRows.Next() {
		var l Link
		if err := lkRows.Scan(&l.A, &l.B); err != nil {
			lkRows.Close()
			return p, err
		}
		if included[l.A] && included[l.B] {
			p.Links = append(p.Links, l)
		}
	}
	lkRows.Close()
	return p, lkRows.Err()
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
