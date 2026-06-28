package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Live SCM status for NATIVE milestones: when a user links an item to a GitHub/
// Gitea pull request, issue or release via scm_url, we periodically read that
// resource's live state and reflect it in the item's progress (and colour). This
// only touches native items (source_system IS NULL) that actually carry a link.

// parseSCMURL turns a PR/issue/release web URL into the API config + resource ref.
func parseSCMURL(raw string) (cfg ghConfig, kind, ref string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ghConfig{}, "", "", false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return ghConfig{}, "", "", false
	}
	cfg.owner = parts[0]
	cfg.repo = strings.TrimSuffix(parts[1], ".git")
	switch parts[2] {
	case "pull", "pulls":
		kind, ref = "pull", parts[3]
	case "issue", "issues":
		kind, ref = "issue", parts[3]
	case "releases":
		if len(parts) >= 5 && parts[3] == "tag" {
			kind, ref = "release", parts[4]
		} else {
			return ghConfig{}, "", "", false
		}
	default:
		return ghConfig{}, "", "", false
	}
	if strings.Contains(u.Host, "github.com") {
		cfg.provider, cfg.apiBase = "github", "" // api.github.com
	} else {
		cfg.provider, cfg.apiBase = "gitea", fmt.Sprintf("%s://%s/api/v1", u.Scheme, u.Host)
	}
	return cfg, kind, ref, true
}

// fetchSCMState reads one resource and maps its state to progress + colour.
func fetchSCMState(ctx context.Context, cfg ghConfig, kind, ref string) (progress *int, color *string, status string, err error) {
	switch kind {
	case "pull":
		var pr struct {
			State    string `json:"state"`
			Merged   bool   `json:"merged"`
			MergedAt string `json:"merged_at"`
		}
		if err = ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/pulls/%s", cfg.owner, cfg.repo, ref), &pr); err != nil {
			return nil, nil, "", err
		}
		switch {
		case pr.Merged || pr.MergedAt != "":
			return ghIntPtr(100), gitColorPtr(cfg.colors.PRMerged), "merged", nil
		case pr.State == "closed":
			return ghIntPtr(0), gitColorPtr(cfg.colors.PRClosed), "closed", nil
		default:
			return ghIntPtr(50), gitColorPtr(cfg.colors.PROpen), "open", nil
		}
	case "issue":
		var is struct {
			State string `json:"state"`
		}
		if err = ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/issues/%s", cfg.owner, cfg.repo, ref), &is); err != nil {
			return nil, nil, "", err
		}
		if is.State == "closed" {
			return ghIntPtr(100), gitColorPtr(cfg.colors.IssueClosed), "closed", nil
		}
		return ghIntPtr(0), gitColorPtr(cfg.colors.IssueOpen), "open", nil
	case "release":
		var rel struct {
			Prerelease bool `json:"prerelease"`
		}
		if err = ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/releases/tags/%s", cfg.owner, cfg.repo, ref), &rel); err != nil {
			return nil, nil, "", err
		}
		if rel.Prerelease {
			return ghIntPtr(100), gitColorPtr(cfg.colors.ReleasePre), "prerelease", nil
		}
		return ghIntPtr(100), gitColorPtr(cfg.colors.ReleaseStable), "released", nil
	}
	return nil, nil, "", fmt.Errorf("unsupported SCM link")
}

// refreshItemSCM polls one native item's linked resource and writes back the
// derived progress/colour. On a fetch error it only bumps scm_checked_at so a
// 404 / rate-limit doesn't retry every tick. Reuses a configured source's token
// for the same repo when one exists (otherwise an anonymous request).
func (s *Store) refreshItemSCM(ctx context.Context, ws, id, scmURL string) (string, error) {
	cfg, kind, ref, ok := parseSCMURL(scmURL)
	if !ok {
		// Not a recognised PR/issue/release link — nothing to track, but stamp it
		// so the sweep doesn't keep reconsidering the same row.
		_, _ = s.pool.Exec(ctx, `UPDATE item SET scm_checked_at = now() WHERE id = $1 AND workspace_id = $2`, id, ws)
		return "", nil
	}
	var tok string
	_ = s.pool.QueryRow(ctx,
		`SELECT token FROM github_source WHERE workspace_id = $1 AND lower(owner) = lower($2) AND lower(repo) = lower($3) LIMIT 1`,
		ws, cfg.owner, cfg.repo).Scan(&tok)
	cfg.token = decToken(tok)
	cfg.colors, _ = s.GetGitColors(ctx, ws)

	progress, color, status, err := fetchSCMState(ctx, cfg, kind, ref)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE item SET scm_checked_at = now() WHERE id = $1 AND workspace_id = $2`, id, ws)
		return "", err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE item SET progress = $3, color = COALESCE($4, color), scm_checked_at = now()
		 WHERE id = $1 AND workspace_id = $2 AND source_system IS NULL`,
		id, ws, progress, color)
	return status, err
}

// RefreshItemSCM refreshes a single native item on demand (API endpoint).
func (s *Store) RefreshItemSCM(ctx context.Context, ws, id string) (string, error) {
	var scm, src *string
	err := s.pool.QueryRow(ctx, `SELECT scm_url, source_system FROM item WHERE id = $1 AND workspace_id = $2`, id, ws).Scan(&scm, &src)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if src != nil {
		return "", fmt.Errorf("synced items refresh from their source, not their SCM link")
	}
	if scm == nil || *scm == "" {
		return "", nil
	}
	return s.refreshItemSCM(ctx, ws, id, *scm)
}

// RefreshDueSCM is the background sweep: it refreshes native items whose link
// hasn't been polled recently. Bounded per tick and throttled to keep anonymous
// API usage well under the rate limit.
func (s *Store) RefreshDueSCM(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT workspace_id, id, scm_url FROM item
		 WHERE source_system IS NULL AND scm_url IS NOT NULL AND scm_url <> ''
		   AND (scm_checked_at IS NULL OR scm_checked_at < now() - interval '20 minutes')
		 ORDER BY scm_checked_at NULLS FIRST
		 LIMIT 25`)
	if err != nil {
		return
	}
	type due struct{ ws, id, scm string }
	var list []due
	for rows.Next() {
		var d due
		if rows.Scan(&d.ws, &d.id, &d.scm) == nil {
			list = append(list, d)
		}
	}
	rows.Close()
	for _, d := range list {
		_, _ = s.refreshItemSCM(ctx, d.ws, d.id, d.scm)
	}
}
