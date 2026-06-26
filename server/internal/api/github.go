package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tpasson/sw-atlas/server/internal/store"
)

// createGitHubSource binds a github.com/owner/repo URL as a read-only source and
// runs a first sync. Failures surface in the source's last_status.
func (s *Server) createGitHubSource(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL             string `json:"url"`
		Token           string `json:"token"`
		IncludeReleases bool   `json:"includeReleases"`
		IncludeTags     bool   `json:"includeTags"`
		IncludeIssues   bool   `json:"includeIssues"`
		IncludePrs      bool   `json:"includePrs"`
		StableOnly      bool   `json:"stableOnly"`
		StateFilter     string `json:"stateFilter"`
		SinceDate       string `json:"sinceDate"`
		MaxPerType      int    `json:"maxPerType"`
	}
	if !decode(w, r, &in) {
		return
	}
	owner, repo, htmlURL, provider, apiBase, err := store.ParseRepoURL(in.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	// Need at least one resource type; default to releases.
	if !in.IncludeReleases && !in.IncludeTags && !in.IncludeIssues && !in.IncludePrs {
		in.IncludeReleases = true
	}
	state := in.StateFilter
	if state != "open" && state != "closed" {
		state = "all"
	}
	if in.MaxPerType < 0 {
		in.MaxPerType = 0
	}

	ws := s.currentWorkspace(r)
	src, err := s.store.CreateGitHubSource(r.Context(), ws, uuid.NewString(), store.GitHubSourceInput{
		Owner: owner, Repo: repo, HTMLURL: htmlURL, Provider: provider, APIBase: apiBase, Token: in.Token,
		Releases: in.IncludeReleases, Tags: in.IncludeTags, Issues: in.IncludeIssues, PRs: in.IncludePrs,
		StableOnly: in.StableOnly, StateFilter: state, SinceDate: strings.TrimSpace(in.SinceDate), MaxPerType: in.MaxPerType,
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	_ = s.store.SyncGitHubSource(r.Context(), ws, src.ID)
	out, err := s.store.GetGitHubSource(r.Context(), ws, src.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) listGitHubSources(w http.ResponseWriter, r *http.Request) {
	srcs, err := s.store.ListGitHubSources(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": srcs})
}

func (s *Server) syncGitHubSource(w http.ResponseWriter, r *http.Request) {
	ws := s.currentWorkspace(r)
	id := chi.URLParam(r, "id")
	if err := s.store.SyncGitHubSource(r.Context(), ws, id); err != nil {
		if err == store.ErrNotFound {
			s.fail(w, err)
			return
		}
		// A fetch error is recorded in last_status; still return the source.
	}
	out, err := s.store.GetGitHubSource(r.Context(), ws, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) setGitHubSourceToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	ws := s.currentWorkspace(r)
	id := chi.URLParam(r, "id")
	if err := s.store.SetGitHubSourceToken(r.Context(), ws, id, strings.TrimSpace(in.Token)); err != nil {
		s.fail(w, err)
		return
	}
	// re-sync with the new token so the result is immediate
	_ = s.store.SyncGitHubSource(r.Context(), ws, id)
	out, err := s.store.GetGitHubSource(r.Context(), ws, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteGitHubSource(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteGitHubSource(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}
