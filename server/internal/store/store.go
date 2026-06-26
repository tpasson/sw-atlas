// Package store is the typed data-access layer over PostgreSQL (pgx). It exposes
// CRUD for swimlanes, sub-lanes, items and links plus app settings, and enforces
// the "source is master" rule: items that originate from an external source
// cannot be modified or deleted through ATLAS.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrLocked   = errors.New("item is managed by an external source and is read-only")
)

// DefaultWorkspaceID is the id of the single workspace that holds all data in
// Slice A of multi-tenancy. Use it instead of bare "default" string literals.
const DefaultWorkspaceID = "default"

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type SubLane struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Swimlane struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	SubLanes     []SubLane `json:"subLanes"`
	SourceSystem *string   `json:"sourceSystem"`         // set => mirrored from a source (read-only)
	SourceKind   *string   `json:"sourceKind,omitempty"` // github | gitea | gitlab | subscription | …
	Hidden       bool      `json:"hidden"`               // consumer-local: hidden from the board
}

// Item is a milestone or event. JSON keys mirror the existing frontend store so
// the API is a drop-in replacement for the old localStorage shape.
type Item struct {
	ID           string  `json:"id"`
	SwimlaneID   string  `json:"swimlaneId"`
	SubLaneID    *string `json:"subLaneId"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	Title        string  `json:"title"`
	What         string  `json:"what"`
	Why          string  `json:"why"`
	How          string  `json:"how"`
	Who          string  `json:"who"`
	When         *string `json:"when"`
	Kind         string  `json:"kind"`
	Marker       string  `json:"marker"`
	StartDate    *string `json:"startDate"`
	EndDate      *string `json:"endDate"`
	Color        *string `json:"color"`
	SourceSystem *string `json:"sourceSystem"`
	ExternalID   *string `json:"externalId"`
	ExternalURL  *string `json:"externalUrl"`
	LastSyncedAt *string `json:"lastSyncedAt"`
	Maturity     *int    `json:"maturity"` // 1..4 (Concept·Design·Production·Series) or null
	Progress     *int    `json:"progress"` // 0..100 (% complete) or null
	ScmURL       *string `json:"scmUrl"`   // link to a source-control resource (release/PR/branch/commit) or null
}

type Link struct {
	A string `json:"a"`
	B string `json:"b"`
}

type Plan struct {
	Swimlanes  []Swimlane `json:"swimlanes"`
	Milestones []Item     `json:"milestones"`
	Links      []Link     `json:"links"`
}

const itemColumns = `id, swimlane_id, sub_lane_id, year, month, title, what, why, how, who,
	when_date, kind, marker, start_date, end_date, color,
	source_system, external_id, external_url, last_synced_at, maturity, progress, scm_url`

// ── Plan (read) ─────────────────────────────────────────────────────────────

func (s *Store) GetPlan(ctx context.Context, ws string) (Plan, error) {
	p := Plan{Swimlanes: []Swimlane{}, Milestones: []Item{}, Links: []Link{}}

	swRows, err := s.pool.Query(ctx, `SELECT id, name, color, source_system, source_kind, hidden FROM swimlane WHERE workspace_id = $1 ORDER BY sort_order, name`, ws)
	if err != nil {
		return p, err
	}
	idx := map[string]int{}
	for swRows.Next() {
		var sw Swimlane
		if err := swRows.Scan(&sw.ID, &sw.Name, &sw.Color, &sw.SourceSystem, &sw.SourceKind, &sw.Hidden); err != nil {
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

	subRows, err := s.pool.Query(ctx, `SELECT id, swimlane_id, name FROM sub_lane WHERE workspace_id = $1 ORDER BY sort_order, name`, ws)
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

	itRows, err := s.pool.Query(ctx, `SELECT `+itemColumns+` FROM item WHERE workspace_id = $1 ORDER BY year, month, title`, ws)
	if err != nil {
		return p, err
	}
	for itRows.Next() {
		it, err := scanItem(itRows)
		if err != nil {
			itRows.Close()
			return p, err
		}
		p.Milestones = append(p.Milestones, it)
	}
	itRows.Close()
	if err := itRows.Err(); err != nil {
		return p, err
	}

	lkRows, err := s.pool.Query(ctx, `SELECT a_item_id, b_item_id FROM link WHERE workspace_id = $1`, ws)
	if err != nil {
		return p, err
	}
	for lkRows.Next() {
		var l Link
		if err := lkRows.Scan(&l.A, &l.B); err != nil {
			lkRows.Close()
			return p, err
		}
		p.Links = append(p.Links, l)
	}
	lkRows.Close()
	return p, lkRows.Err()
}

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	var sub, color, src, extID, extURL, scm *string
	var maturity, progress *int
	var when, start, end, last sql.NullTime
	if err := row.Scan(
		&it.ID, &it.SwimlaneID, &sub, &it.Year, &it.Month,
		&it.Title, &it.What, &it.Why, &it.How, &it.Who,
		&when, &it.Kind, &it.Marker, &start, &end, &color,
		&src, &extID, &extURL, &last, &maturity, &progress, &scm,
	); err != nil {
		return it, err
	}
	it.SubLaneID, it.Color, it.SourceSystem, it.ExternalID, it.ExternalURL = sub, color, src, extID, extURL
	it.Maturity, it.Progress, it.ScmURL = maturity, progress, scm
	it.When = dateStr(when)
	it.StartDate = dateStr(start)
	it.EndDate = dateStr(end)
	it.LastSyncedAt = tsStr(last)
	return it, nil
}

// ── Import ──────────────────────────────────────────────────────────────────

// ImportSummary reports how many entities an import created.
type ImportSummary struct {
	Swimlanes int `json:"swimlanes"`
	SubLanes  int `json:"subLanes"`
	Items     int `json:"items"`
	Links     int `json:"links"`
}

// ImportPlan additively imports a plan (the shared wire format) into the current
// plan inside one transaction. All IDs are remapped to fresh UUIDs so an import
// never collides with existing data, and provenance is stripped so imported
// items become native, editable items (Copy-mode — distinct from a live mirror).
// Items referencing a swimlane absent from the payload, and links with a missing
// endpoint, are skipped rather than failing the whole import.
func (s *Store) ImportPlan(ctx context.Context, ws string, p Plan) (ImportSummary, error) {
	var sum ImportSummary
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sum, err
	}
	defer tx.Rollback(ctx)

	swID := make(map[string]string)  // old swimlane id → new
	subID := make(map[string]string) // old sub-lane id → new
	itID := make(map[string]string)  // old item id → new

	for _, sw := range p.Swimlanes {
		nid := uuid.NewString()
		swID[sw.ID] = nid
		color := sw.Color
		if color == "" {
			color = "#0A84FF"
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO swimlane (id, name, color, sort_order, workspace_id)
			 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM swimlane WHERE workspace_id = $4), $4)`,
			nid, sw.Name, color, ws); err != nil {
			return sum, err
		}
		sum.Swimlanes++
		for _, sub := range sw.SubLanes {
			nsid := uuid.NewString()
			subID[sub.ID] = nsid
			if _, err := tx.Exec(ctx,
				`INSERT INTO sub_lane (id, swimlane_id, name, sort_order, workspace_id)
				 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM sub_lane WHERE workspace_id = $4 AND swimlane_id = $2), $4)`,
				nsid, nid, sub.Name, ws); err != nil {
				return sum, err
			}
			sum.SubLanes++
		}
	}

	for _, it := range p.Milestones {
		nsw, ok := swID[it.SwimlaneID]
		if !ok {
			continue // item's swimlane is not part of this payload
		}
		var nsub interface{}
		if it.SubLaneID != nil {
			if v, ok := subID[*it.SubLaneID]; ok {
				nsub = v
			}
		}
		defaultsForItem(&it)
		whenV, startV, endV, err := itemDates(it)
		if err != nil {
			return sum, err
		}
		nid := uuid.NewString()
		itID[it.ID] = nid
		if _, err := tx.Exec(ctx,
			`INSERT INTO item (`+itemColumns+`, workspace_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
			nid, nsw, nsub, it.Year, it.Month, it.Title, it.What, it.Why, it.How, it.Who,
			whenV, it.Kind, it.Marker, startV, endV, it.Color,
			nil, nil, nil, nil, it.Maturity, it.Progress, it.ScmURL, ws); err != nil { // provenance stripped → native item
			return sum, err
		}
		sum.Items++
	}

	for _, l := range p.Links {
		na, ok1 := itID[l.A]
		nb, ok2 := itID[l.B]
		if !ok1 || !ok2 || na == nb {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO link (a_item_id, b_item_id, workspace_id)
			 SELECT $1, $2, $3
			 WHERE NOT EXISTS (
			   SELECT 1 FROM link WHERE workspace_id = $3 AND ((a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1))
			 )`,
			na, nb, ws); err != nil {
			return sum, err
		}
		sum.Links++
	}

	if err := tx.Commit(ctx); err != nil {
		return sum, err
	}
	return sum, nil
}

// ── Swimlanes ───────────────────────────────────────────────────────────────

func (s *Store) CreateSwimlane(ctx context.Context, ws, id, name, color string) (Swimlane, error) {
	if color == "" {
		color = "#0A84FF"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO swimlane (id, name, color, sort_order, workspace_id)
		 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM swimlane WHERE workspace_id = $4), $4)`,
		id, name, color, ws)
	if err != nil {
		return Swimlane{}, err
	}
	return Swimlane{ID: id, Name: name, Color: color, SubLanes: []SubLane{}}, nil
}

func (s *Store) UpdateSwimlane(ctx context.Context, ws, id string, name, color *string) error {
	// A mirrored lane's name belongs to its source, but the consumer may recolour
	// it locally (the colour then survives re-sync). So block only a name change.
	src, err := s.swimlaneSource(ctx, ws, id)
	if err != nil {
		return err
	}
	if src != nil && name != nil {
		return ErrLocked
	}
	ct, err := s.pool.Exec(ctx,
		`UPDATE swimlane SET name = COALESCE($2, name), color = COALESCE($3, color) WHERE id = $1 AND workspace_id = $4`,
		id, name, color, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSwimlane(ctx context.Context, ws, id string) error {
	if err := s.ensureSwimlaneUnlocked(ctx, ws, id); err != nil {
		return err
	}
	ct, err := s.pool.Exec(ctx, `DELETE FROM swimlane WHERE id = $1 AND workspace_id = $2`, id, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// swimlaneSource returns a lane's source_system (nil for a native lane),
// or ErrNotFound if the lane doesn't exist in this workspace.
func (s *Store) swimlaneSource(ctx context.Context, ws, id string) (*string, error) {
	var src *string
	err := s.pool.QueryRow(ctx, `SELECT source_system FROM swimlane WHERE id = $1 AND workspace_id = $2`, id, ws).Scan(&src)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return src, err
}

// ensureSwimlaneUnlocked rejects edits to mirrored (external) swimlanes — like
// items, the source is master. Native lanes pass.
func (s *Store) ensureSwimlaneUnlocked(ctx context.Context, ws, id string) error {
	src, err := s.swimlaneSource(ctx, ws, id)
	if err != nil {
		return err
	}
	if src != nil {
		return ErrLocked
	}
	return nil
}

// SetSwimlaneHidden toggles a lane's consumer-local visibility (allowed on
// external lanes — it's a local view preference, not a change to the source).
func (s *Store) SetSwimlaneHidden(ctx context.Context, ws, id string, hidden bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE swimlane SET hidden = $2 WHERE id = $1 AND workspace_id = $3`, id, hidden, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MoveSwimlane swaps the sort order of a swimlane with its neighbour (dir -1 up, +1 down).
func (s *Store) MoveSwimlane(ctx context.Context, ws, id string, dir int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var cur int
	if err := tx.QueryRow(ctx, `SELECT sort_order FROM swimlane WHERE id = $1 AND workspace_id = $2`, id, ws).Scan(&cur); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	q := `SELECT id, sort_order FROM swimlane WHERE workspace_id = $2 AND sort_order > $1 ORDER BY sort_order ASC LIMIT 1`
	if dir < 0 {
		q = `SELECT id, sort_order FROM swimlane WHERE workspace_id = $2 AND sort_order < $1 ORDER BY sort_order DESC LIMIT 1`
	}
	var nbID string
	var nbOrder int
	if err := tx.QueryRow(ctx, q, cur, ws).Scan(&nbID, &nbOrder); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already at the edge: no-op
		}
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE swimlane SET sort_order = $2 WHERE id = $1 AND workspace_id = $3`, id, nbOrder, ws); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE swimlane SET sort_order = $2 WHERE id = $1 AND workspace_id = $3`, nbID, cur, ws); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReorderSwimlanes sets sort_order to match the given id order (drag & drop).
// Ids not present are left untouched; the client sends the full ordered list.
func (s *Store) ReorderSwimlanes(ctx context.Context, ws string, ids []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE swimlane SET sort_order = $2 WHERE id = $1 AND workspace_id = $3`, id, i, ws); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ── Sub-lanes ───────────────────────────────────────────────────────────────

func (s *Store) CreateSubLane(ctx context.Context, ws, swimlaneID, id, name string) (SubLane, error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sub_lane (id, swimlane_id, name, sort_order, workspace_id)
		 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM sub_lane WHERE workspace_id = $4 AND swimlane_id = $2), $4)`,
		id, swimlaneID, name, ws)
	if err != nil {
		return SubLane{}, err
	}
	return SubLane{ID: id, Name: name}, nil
}

func (s *Store) UpdateSubLane(ctx context.Context, ws, id, name string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE sub_lane SET name = $2 WHERE id = $1 AND workspace_id = $3`, id, name, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSubLane(ctx context.Context, ws, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM sub_lane WHERE id = $1 AND workspace_id = $2`, id, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderSubLanes sets sort_order to match the given id order (drag & drop within
// one swimlane). The client sends the full ordered list of that lane's sub-lanes.
func (s *Store) ReorderSubLanes(ctx context.Context, ws string, ids []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE sub_lane SET sort_order = $2 WHERE id = $1 AND workspace_id = $3`, id, i, ws); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ── Items ───────────────────────────────────────────────────────────────────

func (s *Store) CreateItem(ctx context.Context, ws string, it Item) (Item, error) {
	// A mirrored (read-only) swimlane only gets its items from its source.
	if err := s.ensureSwimlaneUnlocked(ctx, ws, it.SwimlaneID); err != nil {
		return it, err
	}
	defaultsForItem(&it)
	whenV, startV, endV, err := itemDates(it)
	if err != nil {
		return it, err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO item (`+itemColumns+`, workspace_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		it.ID, it.SwimlaneID, it.SubLaneID, it.Year, it.Month, it.Title, it.What, it.Why, it.How, it.Who,
		whenV, it.Kind, it.Marker, startV, endV, it.Color,
		it.SourceSystem, it.ExternalID, it.ExternalURL, nil, it.Maturity, it.Progress, it.ScmURL, ws)
	if err != nil {
		return it, err
	}
	return it, nil
}

// UpdateItem updates the editable fields of a native item. Items with a
// source_system are rejected with ErrLocked (the source is master). Provenance
// columns are never changed here.
func (s *Store) UpdateItem(ctx context.Context, ws, id string, it Item) error {
	if err := s.ensureUnlocked(ctx, ws, id); err != nil {
		return err
	}
	defaultsForItem(&it)
	whenV, startV, endV, err := itemDates(it)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE item SET
		   swimlane_id=$2, sub_lane_id=$3, year=$4, month=$5, title=$6,
		   what=$7, why=$8, how=$9, who=$10, when_date=$11,
		   kind=$12, marker=$13, start_date=$14, end_date=$15, color=$16, maturity=$17, progress=$18, scm_url=$19
		 WHERE id=$1 AND workspace_id=$20`,
		id, it.SwimlaneID, it.SubLaneID, it.Year, it.Month, it.Title,
		it.What, it.Why, it.How, it.Who, whenV,
		it.Kind, it.Marker, startV, endV, it.Color, it.Maturity, it.Progress, it.ScmURL, ws)
	return err
}

func (s *Store) DeleteItem(ctx context.Context, ws, id string) error {
	if err := s.ensureUnlocked(ctx, ws, id); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM item WHERE id = $1 AND workspace_id = $2`, id, ws)
	return err
}

func (s *Store) ensureUnlocked(ctx context.Context, ws, id string) error {
	var src *string
	err := s.pool.QueryRow(ctx, `SELECT source_system FROM item WHERE id = $1 AND workspace_id = $2`, id, ws).Scan(&src)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if src != nil {
		return ErrLocked
	}
	return nil
}

func defaultsForItem(it *Item) {
	if it.Kind == "" {
		it.Kind = "milestone"
	}
	if it.Marker == "" {
		it.Marker = "l:Diamond"
	}
}

func itemDates(it Item) (whenV, startV, endV interface{}, err error) {
	if whenV, err = toDate(it.When); err != nil {
		return
	}
	if startV, err = toDate(it.StartDate); err != nil {
		return
	}
	endV, err = toDate(it.EndDate)
	return
}

// ── Links ───────────────────────────────────────────────────────────────────

func (s *Store) AddLink(ctx context.Context, ws, a, b string) error {
	if a == b {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO link (a_item_id, b_item_id, workspace_id)
		 SELECT $1, $2, $3
		 WHERE NOT EXISTS (
		   SELECT 1 FROM link WHERE workspace_id = $3 AND ((a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1))
		 )`,
		a, b, ws)
	return err
}

func (s *Store) RemoveLink(ctx context.Context, ws, a, b string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM link WHERE workspace_id = $3 AND ((a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1))`,
		a, b, ws)
	return err
}

// ── Settings ────────────────────────────────────────────────────────────────

func (s *Store) GetPublicRead(ctx context.Context, ws string) (bool, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'public_read_enabled' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

func (s *Store) SetPublicRead(ctx context.Context, ws string, enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'public_read_enabled', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, v)
	return err
}

// GetPalette returns the shared, editor-managed list of custom area colours.
func (s *Store) GetPalette(ctx context.Context, ws string) ([]string, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'area_palette' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // never configured → caller seeds defaults
	}
	if err != nil {
		return nil, err
	}
	var colors []string
	if err := json.Unmarshal([]byte(v), &colors); err != nil {
		return nil, nil
	}
	return colors, nil
}

// SetPalette replaces the shared custom area-colour palette.
func (s *Store) SetPalette(ctx context.Context, ws string, colors []string) error {
	if colors == nil {
		colors = []string{}
	}
	b, err := json.Marshal(colors)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'area_palette', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, string(b))
	return err
}

// GetUISettings returns the per-workspace display settings as an opaque JSON blob
// (nil if never set — the client falls back to its built-in defaults). The server
// doesn't interpret the shape; it's the frontend's settings object.
func (s *Store) GetUISettings(ctx context.Context, ws string) (json.RawMessage, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'ui_settings' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(v), nil
}

// SetUISettings stores the per-workspace display settings (an opaque JSON object).
func (s *Store) SetUISettings(ctx context.Context, ws string, raw json.RawMessage) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	if !json.Valid(raw) {
		return errors.New("settings must be a JSON object")
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'ui_settings', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, string(raw))
	return err
}

// Group is a named, colour-coded collection of items (stored as JSON in app_setting).
type Group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Color   string   `json:"color"`
	ItemIDs []string `json:"itemIds"`
}

// GetGroups returns the shared item groups.
func (s *Store) GetGroups(ctx context.Context, ws string) ([]Group, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'groups' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return []Group{}, nil
	}
	if err != nil {
		return nil, err
	}
	var groups []Group
	if err := json.Unmarshal([]byte(v), &groups); err != nil {
		return []Group{}, nil
	}
	return groups, nil
}

// SetGroups replaces the shared item groups.
func (s *Store) SetGroups(ctx context.Context, ws string, groups []Group) error {
	if groups == nil {
		groups = []Group{}
	}
	b, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'groups', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, string(b))
	return err
}

// CountSwimlanes is used by the seed command to avoid double-seeding.
func (s *Store) CountSwimlanes(ctx context.Context, ws string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM swimlane WHERE workspace_id = $1`, ws).Scan(&n)
	return n, err
}

// ── helpers ─────────────────────────────────────────────────────────────────

func dateStr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format("2006-01-02")
	return &s
}

func tsStr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func toDate(s *string) (interface{}, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return t, nil
}
