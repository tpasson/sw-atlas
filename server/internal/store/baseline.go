package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Baseline is a named snapshot of the plan's items, used to compare against the
// live plan.
type Baseline struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	CreatedAt string `json:"createdAt"`
	ItemCount int    `json:"itemCount"`
	Items     []Item `json:"items,omitempty"`
}

// CreateBaseline captures all current items into a new baseline.
func (s *Store) CreateBaseline(ctx context.Context, ws, id, name, note string) (Baseline, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Baseline{}, err
	}
	defer tx.Rollback(ctx)

	var createdAt time.Time
	if err := tx.QueryRow(ctx,
		`INSERT INTO baseline (id, name, note, workspace_id) VALUES ($1, $2, $3, $4) RETURNING created_at`,
		id, name, note, ws).Scan(&createdAt); err != nil {
		return Baseline{}, err
	}
	ct, err := tx.Exec(ctx,
		`INSERT INTO baseline_item
		   (baseline_id, item_id, swimlane_id, sub_lane_id, title, year, month, when_date, start_date, end_date, kind, marker)
		 SELECT $1, id, swimlane_id, sub_lane_id, title, year, month, when_date, start_date, end_date, kind, marker
		 FROM item WHERE workspace_id = $2`, id, ws)
	if err != nil {
		return Baseline{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Baseline{}, err
	}
	return Baseline{ID: id, Name: name, Note: note, CreatedAt: createdAt.Format(time.RFC3339), ItemCount: int(ct.RowsAffected())}, nil
}

func (s *Store) ListBaselines(ctx context.Context, ws string) ([]Baseline, error) {
	out := []Baseline{}
	rows, err := s.pool.Query(ctx,
		`SELECT b.id, b.name, b.note, b.created_at, COUNT(bi.item_id)
		 FROM baseline b LEFT JOIN baseline_item bi ON bi.baseline_id = b.id
		 WHERE b.workspace_id = $1
		 GROUP BY b.id, b.name, b.note, b.created_at
		 ORDER BY b.created_at DESC`, ws)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var b Baseline
		var ca time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Note, &ca, &b.ItemCount); err != nil {
			return out, err
		}
		b.CreatedAt = ca.Format(time.RFC3339)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) GetBaseline(ctx context.Context, ws, id string) (Baseline, error) {
	var b Baseline
	var ca time.Time
	err := s.pool.QueryRow(ctx, `SELECT id, name, note, created_at FROM baseline WHERE id = $1 AND workspace_id = $2`, id, ws).
		Scan(&b.ID, &b.Name, &b.Note, &ca)
	if errors.Is(err, pgx.ErrNoRows) {
		return Baseline{}, ErrNotFound
	}
	if err != nil {
		return Baseline{}, err
	}
	b.CreatedAt = ca.Format(time.RFC3339)
	b.Items = []Item{}

	rows, err := s.pool.Query(ctx,
		`SELECT item_id, swimlane_id, sub_lane_id, year, month, title, when_date, start_date, end_date, kind, marker
		 FROM baseline_item WHERE baseline_id = $1 ORDER BY year, month, title`, id)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var it Item
		var sub *string
		var when, start, end sql.NullTime
		if err := rows.Scan(&it.ID, &it.SwimlaneID, &sub, &it.Year, &it.Month, &it.Title, &when, &start, &end, &it.Kind, &it.Marker); err != nil {
			return b, err
		}
		it.SubLaneID = sub
		it.When = dateStr(when)
		it.StartDate = dateStr(start)
		it.EndDate = dateStr(end)
		b.Items = append(b.Items, it)
	}
	b.ItemCount = len(b.Items)
	return b, rows.Err()
}

func (s *Store) DeleteBaseline(ctx context.Context, ws, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM baseline WHERE id = $1 AND workspace_id = $2`, id, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
