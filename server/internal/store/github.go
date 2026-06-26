package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// GitHubSource is a repository bound as a read-only source. Its releases/tags/
// issues/PRs are mirrored into one swimlane via applyMirror (source_system = ID).
// The token is never serialized to clients.
type GitHubSource struct {
	ID              string  `json:"id"`
	Owner           string  `json:"owner"`
	Repo            string  `json:"repo"`
	HTMLURL         string  `json:"htmlUrl"`
	Provider        string  `json:"provider"` // github | gitea
	IncludeReleases bool    `json:"includeReleases"`
	IncludeTags     bool    `json:"includeTags"`
	IncludeIssues   bool    `json:"includeIssues"`
	IncludePRs      bool    `json:"includePrs"`
	StableOnly      bool    `json:"stableOnly"`   // releases: skip pre-releases
	StateFilter     string  `json:"stateFilter"`  // issues/PRs: all | open | closed
	SinceDate       string  `json:"sinceDate"`    // only items dated on/after (YYYY-MM-DD)
	MaxPerType      int     `json:"maxPerType"`   // keep the N most recent per type (0 = all)
	LastSyncedAt    *string `json:"lastSyncedAt"`
	LastStatus      string  `json:"lastStatus"`
	CreatedAt       string  `json:"createdAt"`
}

const ghColumns = `id, owner, repo, html_url, provider, include_releases, include_tags, include_issues, include_prs, stable_only, state_filter, since_date, max_per_type, last_synced_at, last_status, created_at`

func scanGitHubSource(row pgx.Row) (GitHubSource, error) {
	var g GitHubSource
	var last *time.Time
	var ca time.Time
	if err := row.Scan(&g.ID, &g.Owner, &g.Repo, &g.HTMLURL, &g.Provider,
		&g.IncludeReleases, &g.IncludeTags, &g.IncludeIssues, &g.IncludePRs,
		&g.StableOnly, &g.StateFilter, &g.SinceDate, &g.MaxPerType,
		&last, &g.LastStatus, &ca); err != nil {
		return g, err
	}
	g.CreatedAt = ca.Format(time.RFC3339)
	if last != nil {
		v := last.Format(time.RFC3339)
		g.LastSyncedAt = &v
	}
	return g, nil
}

// GitHubSourceInput carries everything needed to create a source.
type GitHubSourceInput struct {
	Owner, Repo, HTMLURL, Token string
	Provider                    string // github | gitea
	APIBase                     string // explicit REST base for self-hosted (empty = provider default)
	Releases, Tags, Issues, PRs bool
	StableOnly                  bool
	StateFilter                 string
	SinceDate                   string
	MaxPerType                  int
}

func (s *Store) CreateGitHubSource(ctx context.Context, ws, id string, in GitHubSourceInput) (GitHubSource, error) {
	if in.Provider == "" {
		in.Provider = "github"
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO github_source
		   (id, owner, repo, html_url, provider, api_base, token, include_releases, include_tags, include_issues, include_prs,
		    stable_only, state_filter, since_date, max_per_type, workspace_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		id, in.Owner, in.Repo, in.HTMLURL, in.Provider, in.APIBase, encToken(in.Token), in.Releases, in.Tags, in.Issues, in.PRs,
		in.StableOnly, in.StateFilter, in.SinceDate, in.MaxPerType, ws)
	if err != nil {
		return GitHubSource{}, err
	}
	return s.GetGitHubSource(ctx, ws, id)
}

func (s *Store) GetGitHubSource(ctx context.Context, ws, id string) (GitHubSource, error) {
	g, err := scanGitHubSource(s.pool.QueryRow(ctx, `SELECT `+ghColumns+` FROM github_source WHERE id = $1 AND workspace_id = $2`, id, ws))
	if errors.Is(err, pgx.ErrNoRows) {
		return GitHubSource{}, ErrNotFound
	}
	return g, err
}

func (s *Store) ListGitHubSources(ctx context.Context, ws string) ([]GitHubSource, error) {
	out := []GitHubSource{}
	rows, err := s.pool.Query(ctx, `SELECT `+ghColumns+` FROM github_source WHERE workspace_id = $1 ORDER BY created_at DESC`, ws)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		g, err := scanGitHubSource(rows)
		if err != nil {
			return out, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) markGitHubSync(ctx context.Context, ws, id, status string) {
	_, _ = s.pool.Exec(ctx, `UPDATE github_source SET last_status = $2, last_synced_at = now() WHERE id = $1 AND workspace_id = $3`, id, status, ws)
}

// SetGitHubSourceToken updates the stored token (for private / self-hosted repos)
// without recreating the source.
func (s *Store) SetGitHubSourceToken(ctx context.Context, ws, id, token string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE github_source SET token = $2 WHERE id = $1 AND workspace_id = $3`, id, encToken(token), ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SyncGitHubSource fetches the repo's enabled resources from GitHub and mirrors
// them locally. The outcome (or any error) is recorded in last_status.
func (s *Store) SyncGitHubSource(ctx context.Context, ws, id string) error {
	var cfg ghConfig
	err := s.pool.QueryRow(ctx,
		`SELECT owner, repo, html_url, provider, api_base, token, include_releases, include_tags, include_issues, include_prs,
		        stable_only, state_filter, since_date, max_per_type
		 FROM github_source WHERE id = $1 AND workspace_id = $2`, id, ws).
		Scan(&cfg.owner, &cfg.repo, &cfg.htmlURL, &cfg.provider, &cfg.apiBase, &cfg.token, &cfg.releases, &cfg.tags, &cfg.issues, &cfg.prs,
			&cfg.stableOnly, &cfg.stateFilter, &cfg.since, &cfg.maxPerType)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	cfg.token = decToken(cfg.token) // stored encrypted (or legacy plaintext)
	wire, ferr := fetchGitHub(ctx, cfg)
	if ferr != nil {
		s.markGitHubSync(ctx, ws, id, "error: "+ferr.Error())
		return ferr
	}
	if err := s.applyMirror(ctx, ws, id, cfg.htmlURL, cfg.provider, wire); err != nil {
		s.markGitHubSync(ctx, ws, id, "error: "+err.Error())
		return err
	}
	s.markGitHubSync(ctx, ws, id, fmt.Sprintf("ok · %d items", len(wire.Milestones)))
	return nil
}

// DeleteGitHubSource removes the source and everything it mirrored.
func (s *Store) DeleteGitHubSource(ctx context.Context, ws, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM swimlane WHERE source_system = $1 AND workspace_id = $2`, id, ws); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM item WHERE source_system = $1 AND workspace_id = $2`, id, ws); err != nil {
		return err
	}
	ct, err := tx.Exec(ctx, `DELETE FROM github_source WHERE id = $1 AND workspace_id = $2`, id, ws)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// ParseRepoURL extracts owner/repo and the provider/API base from a repo URL.
// github.com uses GitHub's API; any other host is treated as a self-hosted Gitea
// (REST base <host>/api/v1) — the other supported provider.
func ParseRepoURL(raw string) (owner, repo, htmlURL, provider, apiBase string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", "", "", fmt.Errorf("URL is required")
	}
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", "", "", "", fmt.Errorf("expected a full https://host/owner/repo URL")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", "", "", fmt.Errorf("URL must look like https://host/owner/repo")
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	htmlURL = fmt.Sprintf("%s://%s/%s/%s", u.Scheme, u.Host, owner, repo)
	if strings.Contains(u.Host, "github.com") {
		provider, apiBase = "github", "" // default api.github.com
	} else {
		provider, apiBase = "gitea", fmt.Sprintf("%s://%s/api/v1", u.Scheme, u.Host)
	}
	return owner, repo, htmlURL, provider, apiBase, nil
}

// ── GitHub REST client + mapping ───────────────────────────────────────────────

type ghConfig struct {
	owner, repo, token, htmlURL string
	provider, apiBase           string // gitea uses a token-scheme header + /api/v1 base
	releases, tags, issues, prs bool
	stableOnly                  bool
	stateFilter                 string // all | open | closed
	since                       string // YYYY-MM-DD
	maxPerType                  int
}

var ghClient = &http.Client{Timeout: 20 * time.Second}

func ghBase(cfg ghConfig) string {
	if cfg.apiBase != "" {
		return strings.TrimRight(cfg.apiBase, "/") // self-hosted (Gitea …)
	}
	if v := strings.TrimRight(os.Getenv("ATLAS_GITHUB_API"), "/"); v != "" {
		return v // overridable for tests / GitHub Enterprise
	}
	return "https://api.github.com"
}

func ghGet(ctx context.Context, cfg ghConfig, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ghBase(cfg)+path, nil)
	if err != nil {
		return err
	}
	if cfg.provider == "gitea" {
		req.Header.Set("Accept", "application/json")
		if cfg.token != "" {
			req.Header.Set("Authorization", "token "+cfg.token)
		}
	} else {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if cfg.token != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.token)
		}
	}
	resp, err := ghClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	switch resp.StatusCode {
	case http.StatusOK:
		return json.Unmarshal(body, out)
	case http.StatusNotFound:
		return fmt.Errorf("repository not found (or private — add a token)")
	case http.StatusUnauthorized:
		return fmt.Errorf("auth failed (check the token)")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden or rate-limited — try later or add a token")
	default:
		return fmt.Errorf("the server returned %d", resp.StatusCode)
	}
}

// fetchGitHub builds a one-swimlane mirror feed (Releases/Tags/Issues/PRs as
// sub-lanes) from the repo's enabled resources.
func fetchGitHub(ctx context.Context, cfg ghConfig) (subWire, error) {
	var subLanes []SubLane
	var items []Item
	relTags := map[string]bool{}

	if cfg.releases {
		rel, tags, err := ghReleases(ctx, cfg)
		if err != nil {
			return subWire{}, err
		}
		relTags = tags
		if len(rel) > 0 {
			subLanes = append(subLanes, SubLane{ID: "releases", Name: "Releases"})
			items = append(items, rel...)
		}
	}
	if cfg.tags {
		t, err := ghTags(ctx, cfg, relTags)
		if err != nil {
			return subWire{}, err
		}
		if len(t) > 0 {
			subLanes = append(subLanes, SubLane{ID: "tags", Name: "Tags"})
			items = append(items, t...)
		}
	}
	if cfg.issues {
		is, err := ghIssues(ctx, cfg)
		if err != nil {
			return subWire{}, err
		}
		if len(is) > 0 {
			subLanes = append(subLanes, SubLane{ID: "issues", Name: "Issues"})
			items = append(items, is...)
		}
	}
	if cfg.prs {
		pr, err := ghPulls(ctx, cfg)
		if err != nil {
			return subWire{}, err
		}
		if len(pr) > 0 {
			subLanes = append(subLanes, SubLane{ID: "prs", Name: "Pull requests"})
			items = append(items, pr...)
		}
	}
	return ghWire(cfg.repo, subLanes, items), nil
}

func ghWire(repo string, subLanes []SubLane, items []Item) subWire {
	var w subWire
	w.Milestones = items
	w.Swimlanes = append(w.Swimlanes, struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Color    string    `json:"color"`
		SubLanes []SubLane `json:"subLanes"`
	}{ID: "repo", Name: repo, Color: "#6E5494", SubLanes: subLanes})
	return w
}

func ghReleases(ctx context.Context, cfg ghConfig) ([]Item, map[string]bool, error) {
	var rels []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		PublishedAt string `json:"published_at"`
		CreatedAt   string `json:"created_at"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/releases?per_page=100&limit=50", cfg.owner, cfg.repo), &rels); err != nil {
		return nil, nil, err
	}
	items := []Item{}
	tagSet := map[string]bool{}
	for _, r := range rels {
		if r.Draft {
			continue
		}
		tagSet[r.TagName] = true // dedup tags against ALL release tags, even filtered-out ones
		if cfg.stableOnly && r.Prerelease {
			continue
		}
		ts := r.PublishedAt
		if ts == "" {
			ts = r.CreatedAt
		}
		when, y, m, ok := ghDate(ts)
		if !ok {
			continue
		}
		title := r.Name
		if title == "" {
			title = r.TagName
		}
		var color *string
		if r.Prerelease {
			color = ghStrPtr("#FF9F0A")
		}
		items = append(items, ghItem("release:"+r.TagName, "releases", title, when, y, m,
			"l:Tag", r.HTMLURL, ghText(r.Body, 4000), r.Author.Login, ghIntPtr(100), ghIntPtr(4), color))
	}
	return ghLimit(items, cfg.since, cfg.maxPerType), tagSet, nil
}

func ghTags(ctx context.Context, cfg ghConfig, skip map[string]bool) ([]Item, error) {
	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA     string `json:"sha"`
			Created string `json:"created"` // Gitea provides the commit date here
		} `json:"commit"`
	}
	if err := ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/tags?per_page=100&limit=50", cfg.owner, cfg.repo), &tags); err != nil {
		return nil, err
	}
	items := []Item{}
	const maxTags = 30 // GitHub needs a commit lookup per tag; bound the rate-limit cost
	n := 0
	for _, t := range tags {
		if skip[t.Name] { // already shown as a release
			continue
		}
		if n >= maxTags {
			break
		}
		n++
		dateStr := t.Commit.Created // Gitea: present already
		if dateStr == "" {
			// GitHub: the tags endpoint has no date — fetch the commit.
			var commit struct {
				Commit struct {
					Committer struct {
						Date string `json:"date"`
					} `json:"committer"`
				} `json:"commit"`
			}
			if err := ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/commits/%s", cfg.owner, cfg.repo, t.Commit.SHA), &commit); err != nil {
				continue
			}
			dateStr = commit.Commit.Committer.Date
		}
		when, y, m, ok := ghDate(dateStr)
		if !ok {
			continue
		}
		scm := fmt.Sprintf("%s/releases/tag/%s", cfg.htmlURL, t.Name)
		items = append(items, ghItem("tag:"+t.Name, "tags", t.Name, when, y, m,
			"l:Tag", scm, "", "", ghIntPtr(100), ghIntPtr(4), nil))
	}
	return ghLimit(items, cfg.since, cfg.maxPerType), nil
}

func ghIssues(ctx context.Context, cfg ghConfig) ([]Item, error) {
	var issues []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		State       string `json:"state"`
		CreatedAt   string `json:"created_at"`
		ClosedAt    string `json:"closed_at"`
		HTMLURL     string `json:"html_url"`
		Body        string `json:"body"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
		PullRequest *struct{} `json:"pull_request"` // present ⇒ it's a PR, not an issue
	}
	ip := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=100&limit=50", cfg.owner, cfg.repo, ghState(cfg))
	if cfg.provider == "gitea" {
		ip += "&type=issues" // Gitea returns PRs on /issues unless filtered
	}
	if err := ghGet(ctx, cfg, ip, &issues); err != nil {
		return nil, err
	}
	items := []Item{}
	for _, is := range issues {
		if is.PullRequest != nil {
			continue
		}
		ts := is.CreatedAt
		if is.State == "closed" && is.ClosedAt != "" {
			ts = is.ClosedAt
		}
		when, y, m, ok := ghDate(ts)
		if !ok {
			continue
		}
		color := ghStrPtr("#3FB950")
		progress := ghIntPtr(0)
		if is.State == "closed" {
			color, progress = ghStrPtr("#8957E5"), ghIntPtr(100)
		}
		items = append(items, ghItem(fmt.Sprintf("issue:%d", is.Number), "issues", is.Title, when, y, m,
			"l:CircleDot", is.HTMLURL, ghText(is.Body, 600), is.User.Login, progress, nil, color))
	}
	return ghLimit(items, cfg.since, cfg.maxPerType), nil
}

func ghPulls(ctx context.Context, cfg ghConfig) ([]Item, error) {
	var prs []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		CreatedAt string `json:"created_at"`
		ClosedAt  string `json:"closed_at"`
		MergedAt  string `json:"merged_at"`
		HTMLURL   string `json:"html_url"`
		Body      string `json:"body"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := ghGet(ctx, cfg, fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=100&limit=50", cfg.owner, cfg.repo, ghState(cfg)), &prs); err != nil {
		return nil, err
	}
	items := []Item{}
	for _, p := range prs {
		ts, merged := p.CreatedAt, p.MergedAt != ""
		switch {
		case merged:
			ts = p.MergedAt
		case p.State == "closed" && p.ClosedAt != "":
			ts = p.ClosedAt
		}
		when, y, m, ok := ghDate(ts)
		if !ok {
			continue
		}
		var color *string
		var progress *int
		switch {
		case merged:
			color, progress = ghStrPtr("#8957E5"), ghIntPtr(100)
		case p.State == "closed":
			color, progress = ghStrPtr("#F85149"), ghIntPtr(0)
		default:
			color, progress = ghStrPtr("#3FB950"), ghIntPtr(50)
		}
		items = append(items, ghItem(fmt.Sprintf("pr:%d", p.Number), "prs", p.Title, when, y, m,
			"l:GitPullRequest", p.HTMLURL, ghText(p.Body, 600), p.User.Login, progress, nil, color))
	}
	return ghLimit(items, cfg.since, cfg.maxPerType), nil
}

func ghItem(id, sub, title, when string, y, m int, marker, scm, what, who string, progress, maturity *int, color *string) Item {
	sl, w, sc := sub, when, scm
	return Item{
		ID: id, SwimlaneID: "repo", SubLaneID: &sl,
		Year: y, Month: m, When: &w, Title: title,
		Kind: "milestone", Marker: marker, ScmURL: &sc,
		What: what, Who: who, Progress: progress, Maturity: maturity, Color: color,
	}
}

func ghDate(ts string) (string, int, int, bool) {
	if ts == "" {
		return "", 0, 0, false
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "", 0, 0, false
	}
	return t.Format("2006-01-02"), t.Year(), int(t.Month()), true
}

// Markdown matchers — strip the markup so a release/issue/PR body reads as plain
// prose. Line breaks are preserved (the modal shows the full multi-line text; the
// board tooltip clamps it with CSS).
var (
	mdComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	mdImage   = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	mdLink    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdHeading = regexp.MustCompile(`(?m)^[ \t]*#{1,6}[ \t]*`)
	mdQuote   = regexp.MustCompile(`(?m)^[ \t]*>[ \t]?`)
	mdEmph    = regexp.MustCompile("(\\*\\*|__|\\*|`|~~)")
	mdSpaces  = regexp.MustCompile(`[ \t]+`)
	mdBlanks  = regexp.MustCompile(`\n{3,}`)
)

func ghText(s string, max int) string {
	s = mdComment.ReplaceAllString(s, "")
	s = mdImage.ReplaceAllString(s, "")
	s = mdLink.ReplaceAllString(s, "$1")
	s = mdHeading.ReplaceAllString(s, "")
	s = mdQuote.ReplaceAllString(s, "")
	s = mdEmph.ReplaceAllString(s, "")
	s = mdSpaces.ReplaceAllString(s, " ")
	s = mdBlanks.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > max {
		return strings.TrimSpace(string(r[:max])) + "…"
	}
	return s
}

func ghIntPtr(v int) *int       { return &v }
func ghStrPtr(v string) *string { return &v }

func ghWhen(it Item) string {
	if it.When != nil {
		return *it.When // YYYY-MM-DD — lexical order == chronological order
	}
	return ""
}

// ghLimit drops items dated before `since` and keeps only the `max` most recent.
func ghLimit(items []Item, since string, max int) []Item {
	if since != "" {
		kept := items[:0:0]
		for _, it := range items {
			if ghWhen(it) >= since {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	if max > 0 && len(items) > max {
		sort.SliceStable(items, func(i, j int) bool { return ghWhen(items[i]) > ghWhen(items[j]) })
		items = items[:max]
	}
	return items
}

func ghState(cfg ghConfig) string {
	switch cfg.stateFilter {
	case "open", "closed":
		return cfg.stateFilter
	default:
		return "all"
	}
}
