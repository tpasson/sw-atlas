package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ChangeRequest is a proposed change to the plan, pending the owner's decision.
type ChangeRequest struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind"` // edit | create
	TargetItemID *string         `json:"targetItemId"`
	TargetTitle  string          `json:"targetTitle"` // current title of the target (display)
	Payload      json.RawMessage `json:"payload"`     // proposed item fields
	Note         string          `json:"note"`
	Status       string          `json:"status"` // pending | approved | rejected
	AuthorID     *string         `json:"authorId"`
	AuthorName   string          `json:"authorName"`
	DecidedBy    *string         `json:"decidedBy"`
	DeciderName  string          `json:"deciderName"`
	DecidedAt    *string         `json:"decidedAt"`
	DecisionNote string          `json:"decisionNote"`
	CreatedAt    string          `json:"createdAt"`
}

const crColumns = `cr.id, cr.kind, cr.target_item_id, cr.payload, cr.note, cr.status,
	cr.author_id, au.username, cr.decided_by, du.username, cr.decided_at, cr.decision_note, cr.created_at,
	COALESCE(it.title, '')`

const crFrom = `FROM change_request cr
	LEFT JOIN app_user au ON au.id = cr.author_id
	LEFT JOIN app_user du ON du.id = cr.decided_by
	LEFT JOIN item it ON it.id = cr.target_item_id AND it.workspace_id = cr.workspace_id`

func scanChangeRequest(row pgx.Row) (ChangeRequest, error) {
	var cr ChangeRequest
	var target, authorName, deciderName *string
	var payload []byte
	var decidedAt sql.NullTime
	var createdAt time.Time
	if err := row.Scan(
		&cr.ID, &cr.Kind, &target, &payload, &cr.Note, &cr.Status,
		&cr.AuthorID, &authorName, &cr.DecidedBy, &deciderName, &decidedAt, &cr.DecisionNote, &createdAt,
		&cr.TargetTitle,
	); err != nil {
		return cr, err
	}
	cr.TargetItemID = target
	cr.Payload = json.RawMessage(payload)
	if authorName != nil {
		cr.AuthorName = *authorName
	}
	if deciderName != nil {
		cr.DeciderName = *deciderName
	}
	cr.DecidedAt = tsStr(decidedAt)
	cr.CreatedAt = createdAt.Format(time.RFC3339)
	return cr, nil
}

// CreateChangeRequest stores a new pending proposal.
func (s *Store) CreateChangeRequest(ctx context.Context, ws, id, authorID, kind, targetItemID string, payload []byte, note string) (ChangeRequest, error) {
	if kind != "create" {
		kind = "edit"
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO change_request (id, workspace_id, author_id, kind, target_item_id, payload, note)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, ws, nullIfEmpty(authorID), kind, nullIfEmpty(targetItemID), payload, note); err != nil {
		return ChangeRequest{}, err
	}
	return s.GetChangeRequest(ctx, ws, id)
}

// ListChangeRequests returns the workspace's change requests (pending first).
func (s *Store) ListChangeRequests(ctx context.Context, ws string) ([]ChangeRequest, error) {
	out := []ChangeRequest{}
	rows, err := s.pool.Query(ctx,
		`SELECT `+crColumns+` `+crFrom+`
		 WHERE cr.workspace_id = $1
		 ORDER BY (cr.status = 'pending') DESC, cr.created_at DESC`, ws)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		cr, err := scanChangeRequest(rows)
		if err != nil {
			return out, err
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}

// GetChangeRequest returns one change request.
func (s *Store) GetChangeRequest(ctx context.Context, ws, id string) (ChangeRequest, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+crColumns+` `+crFrom+` WHERE cr.workspace_id = $1 AND cr.id = $2`, ws, id)
	cr, err := scanChangeRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return cr, ErrNotFound
	}
	return cr, err
}

// ApproveChangeRequest applies the proposed change to the live plan (recording
// the proposer as the change's author) and marks the request approved.
func (s *Store) ApproveChangeRequest(ctx context.Context, ws, id, deciderID, note string) (ChangeRequest, error) {
	cr, err := s.GetChangeRequest(ctx, ws, id)
	if err != nil {
		return cr, err
	}
	if cr.Status != "pending" {
		return cr, errors.New("this change request was already decided")
	}
	var it Item
	if err := json.Unmarshal(cr.Payload, &it); err != nil {
		return cr, err
	}
	actor := ""
	if cr.AuthorID != nil {
		actor = *cr.AuthorID
	}
	itemID := it.ID
	if cr.Kind == "create" {
		if _, err := s.CreateItemAs(ctx, ws, actor, it); err != nil {
			return cr, err
		}
	} else {
		if cr.TargetItemID == nil {
			return cr, errors.New("edit request has no target item")
		}
		itemID = *cr.TargetItemID
		if err := s.UpdateItemAs(ctx, ws, *cr.TargetItemID, actor, it); err != nil {
			return cr, err
		}
	}
	// Apply the proposed dependencies. The payload carries the item's full desired
	// link set (both directions) under "links"; a missing key means the proposal
	// predates link support, so we leave existing links untouched. When present we
	// replace the item's links wholesale so additions AND removals both take effect.
	var lp struct {
		Links *[]struct {
			A           string `json:"a"`
			B           string `json:"b"`
			Rel         string `json:"rel"`
			Version     *int   `json:"version"`
			Qty         *int   `json:"qty"`
			Designators string `json:"designators"`
		} `json:"links"`
	}
	if err := json.Unmarshal(cr.Payload, &lp); err == nil && lp.Links != nil && itemID != "" {
		if _, err := s.pool.Exec(ctx,
			`DELETE FROM link WHERE workspace_id=$1 AND (a_item_id=$2 OR b_item_id=$2)`, ws, itemID); err != nil {
			return cr, err
		}
		for _, l := range *lp.Links {
			if err := s.AddLink(ctx, ws, l.A, l.B, l.Rel, l.Version, l.Qty, l.Designators); err != nil {
				return cr, err
			}
		}
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE change_request SET status='approved', decided_by=$3, decided_at=now(), decision_note=$4
		 WHERE id=$1 AND workspace_id=$2 AND status='pending'`,
		id, ws, nullIfEmpty(deciderID), note); err != nil {
		return cr, err
	}
	return s.GetChangeRequest(ctx, ws, id)
}

// RejectChangeRequest marks a pending request rejected (no change applied).
func (s *Store) RejectChangeRequest(ctx context.Context, ws, id, deciderID, note string) (ChangeRequest, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE change_request SET status='rejected', decided_by=$3, decided_at=now(), decision_note=$4
		 WHERE id=$1 AND workspace_id=$2 AND status='pending'`,
		id, ws, nullIfEmpty(deciderID), note)
	if err != nil {
		return ChangeRequest{}, err
	}
	if ct.RowsAffected() == 0 {
		return ChangeRequest{}, ErrNotFound
	}
	return s.GetChangeRequest(ctx, ws, id)
}
