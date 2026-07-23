package store

import (
	"context"
	"time"
)

// Comment is a free-text note on an item (the Log tab's discussion stream).
// Bodies may reference other items inline as "[[<item-id>]]" tokens; those are
// resolved to titles client-side so they stay current.
type Comment struct {
	ID        string    `json:"id"`
	ItemID    string    `json:"itemId"`
	AuthorID  string    `json:"authorId,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Store) ListComments(ctx context.Context, ws, itemID string) ([]Comment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, item_id, COALESCE(author_id, ''), body, created_at
		   FROM item_comment WHERE workspace_id = $1 AND item_id = $2 ORDER BY created_at`, ws, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ItemID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddComment(ctx context.Context, ws, id, itemID, authorID, body string) (Comment, error) {
	var c Comment
	err := s.pool.QueryRow(ctx,
		`INSERT INTO item_comment (id, workspace_id, item_id, author_id, body)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, item_id, COALESCE(author_id, ''), body, created_at`,
		id, ws, itemID, nullIfEmpty(authorID), body).
		Scan(&c.ID, &c.ItemID, &c.AuthorID, &c.Body, &c.CreatedAt)
	return c, err
}

// ListMentions returns every comment (workspace-wide) whose body references the
// given item as an inline [[<item-id>]] token — the back-link side of comment
// references. The comment's own ItemID tells WHERE the mention was written.
func (s *Store) ListMentions(ctx context.Context, ws, itemID string) ([]Comment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, item_id, COALESCE(author_id, ''), body, created_at
		   FROM item_comment WHERE workspace_id = $1 AND body LIKE '%[[' || $2 || ']]%' ORDER BY created_at`, ws, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ItemID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteComment removes a comment, but only when authorID matches — comments can
// be deleted by their author (there is no moderator delete yet).
func (s *Store) DeleteComment(ctx context.Context, ws, id, authorID string) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM item_comment WHERE workspace_id = $1 AND id = $2 AND author_id = $3`, ws, id, authorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
