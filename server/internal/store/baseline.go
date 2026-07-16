package store

import (
	"context"
	"database/sql"
	"encoding/json"
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
	// Self-heal: ensure every current item has a revision at its current version
	// (covers mirrored items and any gap) so the baseline's pointers resolve.
	if _, err := tx.Exec(ctx,
		`INSERT INTO item_revision (workspace_id, item_id, version, snapshot, edited_by)
		 SELECT workspace_id, id, version,
		        jsonb_build_object(
		          'id', id, 'swimlaneId', swimlane_id, 'subLaneId', sub_lane_id,
		          'year', year, 'month', month, 'title', title,
		          'when', to_char(when_date, 'YYYY-MM-DD'),
		          'kind', kind, 'typeKey', type_key, 'marker', marker,
		          'startDate', to_char(start_date, 'YYYY-MM-DD'),
		          'endDate', to_char(end_date, 'YYYY-MM-DD'),
		          'color', color, 'maturity', maturity, 'progress', progress,
		          'scmUrl', scm_url, 'assigneeId', assignee_id, 'status', status,
		          'sourceSystem', source_system, 'data', data, 'version', version
		        ),
		        updated_by
		 FROM item i
		 WHERE workspace_id = $1
		   AND NOT EXISTS (SELECT 1 FROM item_revision r
		                    WHERE r.workspace_id = i.workspace_id AND r.item_id = i.id AND r.version = i.version)`,
		ws); err != nil {
		return Baseline{}, err
	}
	// A baseline references each item at its current version — no data duplication.
	ct, err := tx.Exec(ctx,
		`INSERT INTO baseline_item (baseline_id, item_id, version)
		 SELECT $1, id, version FROM item WHERE workspace_id = $2`, id, ws)
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

	// Version-pointer rows resolve through item_revision (full snapshot); older
	// rows fall back to the inline snapshot columns captured at the time.
	rows, err := s.pool.Query(ctx,
		`SELECT bi.item_id, bi.version, bi.swimlane_id, bi.sub_lane_id, bi.year, bi.month, bi.title,
		        bi.when_date, bi.start_date, bi.end_date, bi.kind, bi.marker, r.snapshot
		 FROM baseline_item bi
		 LEFT JOIN item_revision r
		        ON r.workspace_id = $2 AND r.item_id = bi.item_id AND r.version = bi.version
		 WHERE bi.baseline_id = $1
		 ORDER BY bi.item_id`, id, ws)
	if err != nil {
		return b, err
	}
	defer rows.Close()
	for rows.Next() {
		var it Item
		var version, year, month *int
		var swl, sub, title, kind, marker *string
		var when, start, end sql.NullTime
		var snap []byte
		if err := rows.Scan(&it.ID, &version, &swl, &sub, &year, &month, &title,
			&when, &start, &end, &kind, &marker, &snap); err != nil {
			return b, err
		}
		if len(snap) > 0 {
			if err := json.Unmarshal(snap, &it); err != nil {
				return b, err
			}
		} else {
			if swl != nil {
				it.SwimlaneID = *swl
			}
			it.SubLaneID = sub
			if year != nil {
				it.Year = *year
			}
			if month != nil {
				it.Month = *month
			}
			if title != nil {
				it.Title = *title
			}
			if kind != nil {
				it.Kind = *kind
			}
			if marker != nil {
				it.Marker = *marker
			}
			it.When = dateStr(when)
			it.StartDate = dateStr(start)
			it.EndDate = dateStr(end)
		}
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
