package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GitColors is the per-workspace colour scheme applied to synced GitHub/Gitea
// items by type/state. An empty field means "inherit the lane's colour".
type GitColors struct {
	ReleaseStable string `json:"releaseStable"`
	ReleasePre    string `json:"releasePre"`
	Tag           string `json:"tag"`
	IssueOpen     string `json:"issueOpen"`
	IssueClosed   string `json:"issueClosed"`
	PROpen        string `json:"prOpen"`
	PRMerged      string `json:"prMerged"`
	PRClosed      string `json:"prClosed"`
}

// DefaultGitColors mirrors the colours ATLAS has always used out of the box.
func DefaultGitColors() GitColors {
	return GitColors{
		ReleaseStable: "", // inherit lane
		ReleasePre:    "#FF9F0A",
		Tag:           "", // inherit lane
		IssueOpen:     "#3FB950",
		IssueClosed:   "#8957E5",
		PROpen:        "#3FB950",
		PRMerged:      "#8957E5",
		PRClosed:      "#F85149",
	}
}

// GetGitColors returns the workspace's synced-item colour scheme, with the
// built-in defaults applied for any field that was never set.
func (s *Store) GetGitColors(ctx context.Context, ws string) (GitColors, error) {
	c := DefaultGitColors()
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'git_colors' AND workspace_id = $1`, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal([]byte(v), &c) // overlay stored values over the defaults
	return c, nil
}

// SetGitColors stores the workspace's synced-item colour scheme.
func (s *Store) SetGitColors(ctx context.Context, ws string, c GitColors) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO app_setting (workspace_id, key, value) VALUES ($1, 'git_colors', $2)
		 ON CONFLICT (workspace_id, key) DO UPDATE SET value = EXCLUDED.value`, ws, string(b))
	return err
}

// gitColorPtr turns a configured colour into an item colour pointer; "" => nil
// (the item inherits its lane's colour).
func gitColorPtr(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}
