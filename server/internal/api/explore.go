package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// explore is the public discovery directory: every publicly-readable plan on the
// server, for the landing page. No auth — anonymous visitors see it.
func (s *Server) explore(w http.ResponseWriter, r *http.Request) {
	plans, err := s.store.ListPublicWorkspaces(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	// Email addresses are only exposed to authenticated requesters.
	if _, ok := s.authedSession(r); !ok {
		for i := range plans {
			plans[i].OwnerEmail = ""
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": plans})
}

// setWorkspaceFeatured pins/unpins a plan on the explore page (admin only).
func (s *Server) setWorkspaceFeatured(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Featured bool `json:"featured"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetWorkspaceFeatured(r.Context(), chi.URLParam(r, "slug"), in.Featured); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"featured": in.Featured})
}
