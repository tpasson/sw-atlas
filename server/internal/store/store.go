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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrLocked   = errors.New("item is managed by an external source and is read-only")
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type SubLane struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Swimlane struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	SubLanes []SubLane `json:"subLanes"`
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
	source_system, external_id, external_url, last_synced_at`

// ── Plan (read) ─────────────────────────────────────────────────────────────

func (s *Store) GetPlan(ctx context.Context) (Plan, error) {
	p := Plan{Swimlanes: []Swimlane{}, Milestones: []Item{}, Links: []Link{}}

	swRows, err := s.pool.Query(ctx, `SELECT id, name, color FROM swimlane ORDER BY sort_order, name`)
	if err != nil {
		return p, err
	}
	idx := map[string]int{}
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

	subRows, err := s.pool.Query(ctx, `SELECT id, swimlane_id, name FROM sub_lane ORDER BY sort_order, name`)
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

	itRows, err := s.pool.Query(ctx, `SELECT `+itemColumns+` FROM item ORDER BY year, month, title`)
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
		p.Links = append(p.Links, l)
	}
	lkRows.Close()
	return p, lkRows.Err()
}

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	var sub, color, src, extID, extURL *string
	var when, start, end, last sql.NullTime
	if err := row.Scan(
		&it.ID, &it.SwimlaneID, &sub, &it.Year, &it.Month,
		&it.Title, &it.What, &it.Why, &it.How, &it.Who,
		&when, &it.Kind, &it.Marker, &start, &end, &color,
		&src, &extID, &extURL, &last,
	); err != nil {
		return it, err
	}
	it.SubLaneID, it.Color, it.SourceSystem, it.ExternalID, it.ExternalURL = sub, color, src, extID, extURL
	it.When = dateStr(when)
	it.StartDate = dateStr(start)
	it.EndDate = dateStr(end)
	it.LastSyncedAt = tsStr(last)
	return it, nil
}

// ── Swimlanes ───────────────────────────────────────────────────────────────

func (s *Store) CreateSwimlane(ctx context.Context, id, name, color string) (Swimlane, error) {
	if color == "" {
		color = "#0A84FF"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO swimlane (id, name, color, sort_order)
		 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM swimlane))`,
		id, name, color)
	if err != nil {
		return Swimlane{}, err
	}
	return Swimlane{ID: id, Name: name, Color: color, SubLanes: []SubLane{}}, nil
}

func (s *Store) UpdateSwimlane(ctx context.Context, id string, name, color *string) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE swimlane SET name = COALESCE($2, name), color = COALESCE($3, color) WHERE id = $1`,
		id, name, color)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSwimlane(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM swimlane WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MoveSwimlane swaps the sort order of a swimlane with its neighbour (dir -1 up, +1 down).
func (s *Store) MoveSwimlane(ctx context.Context, id string, dir int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var cur int
	if err := tx.QueryRow(ctx, `SELECT sort_order FROM swimlane WHERE id = $1`, id).Scan(&cur); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	q := `SELECT id, sort_order FROM swimlane WHERE sort_order > $1 ORDER BY sort_order ASC LIMIT 1`
	if dir < 0 {
		q = `SELECT id, sort_order FROM swimlane WHERE sort_order < $1 ORDER BY sort_order DESC LIMIT 1`
	}
	var nbID string
	var nbOrder int
	if err := tx.QueryRow(ctx, q, cur).Scan(&nbID, &nbOrder); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already at the edge: no-op
		}
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE swimlane SET sort_order = $2 WHERE id = $1`, id, nbOrder); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE swimlane SET sort_order = $2 WHERE id = $1`, nbID, cur); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── Sub-lanes ───────────────────────────────────────────────────────────────

func (s *Store) CreateSubLane(ctx context.Context, swimlaneID, id, name string) (SubLane, error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sub_lane (id, swimlane_id, name, sort_order)
		 VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM sub_lane WHERE swimlane_id = $2))`,
		id, swimlaneID, name)
	if err != nil {
		return SubLane{}, err
	}
	return SubLane{ID: id, Name: name}, nil
}

func (s *Store) UpdateSubLane(ctx context.Context, id, name string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE sub_lane SET name = $2 WHERE id = $1`, id, name)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSubLane(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM sub_lane WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Items ───────────────────────────────────────────────────────────────────

func (s *Store) CreateItem(ctx context.Context, it Item) (Item, error) {
	defaultsForItem(&it)
	whenV, startV, endV, err := itemDates(it)
	if err != nil {
		return it, err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO item (`+itemColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		it.ID, it.SwimlaneID, it.SubLaneID, it.Year, it.Month, it.Title, it.What, it.Why, it.How, it.Who,
		whenV, it.Kind, it.Marker, startV, endV, it.Color,
		it.SourceSystem, it.ExternalID, it.ExternalURL, nil)
	if err != nil {
		return it, err
	}
	return it, nil
}

// UpdateItem updates the editable fields of a native item. Items with a
// source_system are rejected with ErrLocked (the source is master). Provenance
// columns are never changed here.
func (s *Store) UpdateItem(ctx context.Context, id string, it Item) error {
	if err := s.ensureUnlocked(ctx, id); err != nil {
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
		   kind=$12, marker=$13, start_date=$14, end_date=$15, color=$16
		 WHERE id=$1`,
		id, it.SwimlaneID, it.SubLaneID, it.Year, it.Month, it.Title,
		it.What, it.Why, it.How, it.Who, whenV,
		it.Kind, it.Marker, startV, endV, it.Color)
	return err
}

func (s *Store) DeleteItem(ctx context.Context, id string) error {
	if err := s.ensureUnlocked(ctx, id); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM item WHERE id = $1`, id)
	return err
}

func (s *Store) ensureUnlocked(ctx context.Context, id string) error {
	var src *string
	err := s.pool.QueryRow(ctx, `SELECT source_system FROM item WHERE id = $1`, id).Scan(&src)
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
		it.Marker = "diamond"
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

func (s *Store) AddLink(ctx context.Context, a, b string) error {
	if a == b {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO link (a_item_id, b_item_id)
		 SELECT $1, $2
		 WHERE NOT EXISTS (
		   SELECT 1 FROM link WHERE (a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1)
		 )`,
		a, b)
	return err
}

func (s *Store) RemoveLink(ctx context.Context, a, b string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM link WHERE (a_item_id=$1 AND b_item_id=$2) OR (a_item_id=$2 AND b_item_id=$1)`,
		a, b)
	return err
}

// ── Settings ────────────────────────────────────────────────────────────────

func (s *Store) GetPublicRead(ctx context.Context) (bool, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'public_read_enabled'`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

func (s *Store) SetPublicRead(ctx context.Context, enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_setting (key, value) VALUES ('public_read_enabled', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, v)
	return err
}

// GetPalette returns the shared, editor-managed list of custom area colours.
func (s *Store) GetPalette(ctx context.Context) ([]string, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'area_palette'`).Scan(&v)
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
func (s *Store) SetPalette(ctx context.Context, colors []string) error {
	if colors == nil {
		colors = []string{}
	}
	b, err := json.Marshal(colors)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (key, value) VALUES ('area_palette', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, string(b))
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
func (s *Store) GetGroups(ctx context.Context) ([]Group, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'groups'`).Scan(&v)
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
func (s *Store) SetGroups(ctx context.Context, groups []Group) error {
	if groups == nil {
		groups = []Group{}
	}
	b, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (key, value) VALUES ('groups', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, string(b))
	return err
}

// CountSwimlanes is used by the seed command to avoid double-seeding.
func (s *Store) CountSwimlanes(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM swimlane`).Scan(&n)
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
