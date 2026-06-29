package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ItemRevision is one recorded version of an item. Snapshot is the full item
// JSON as it stood at that version (omitted from list responses).
type ItemRevision struct {
	Version  int             `json:"version"`
	EditedBy *string         `json:"editedBy"`
	EditedAt string          `json:"editedAt"`
	Snapshot json.RawMessage `json:"snapshot,omitempty"`
}

// ListItemRevisions returns an item's version history, newest first (no snapshots).
func (s *Store) ListItemRevisions(ctx context.Context, ws, itemID string) ([]ItemRevision, error) {
	out := []ItemRevision{}
	rows, err := s.pool.Query(ctx,
		`SELECT version, edited_by, edited_at FROM item_revision
		 WHERE workspace_id = $1 AND item_id = $2 ORDER BY version DESC`, ws, itemID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var rv ItemRevision
		var at time.Time
		if err := rows.Scan(&rv.Version, &rv.EditedBy, &at); err != nil {
			return out, err
		}
		rv.EditedAt = at.Format(time.RFC3339)
		out = append(out, rv)
	}
	return out, rows.Err()
}

// GetItemRevision returns one version's full snapshot.
func (s *Store) GetItemRevision(ctx context.Context, ws, itemID string, version int) (ItemRevision, error) {
	var rv ItemRevision
	var at time.Time
	var snap []byte
	err := s.pool.QueryRow(ctx,
		`SELECT version, edited_by, edited_at, snapshot FROM item_revision
		 WHERE workspace_id = $1 AND item_id = $2 AND version = $3`, ws, itemID, version).
		Scan(&rv.Version, &rv.EditedBy, &at, &snap)
	if errors.Is(err, pgx.ErrNoRows) {
		return rv, ErrNotFound
	}
	if err != nil {
		return rv, err
	}
	rv.EditedAt = at.Format(time.RFC3339)
	rv.Snapshot = json.RawMessage(snap)
	return rv, nil
}
