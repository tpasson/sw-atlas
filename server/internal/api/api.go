// Package api wires the HTTP routes: public auth endpoints, read endpoints
// gated by the public-read switch, and write endpoints restricted to editors.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/tpasson/sw-atlas/server/internal/auth"
	"github.com/tpasson/sw-atlas/server/internal/store"
)

type Server struct {
	store     *store.Store
	auth      *auth.Auth
	staticDir string
}

func NewRouter(st *store.Store, au *auth.Auth, staticDir string) http.Handler {
	s := &Server{store: st, auth: au, staticDir: staticDir}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.health)
		r.Post("/login", s.login)
		r.Post("/logout", s.logout)
		r.Get("/me", s.me)

		// Read endpoints: open when public-read is on, otherwise editor-only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireReadAccess)
			r.Get("/plan", s.getPlan)
			r.Get("/settings/public-read", s.getPublicRead)
			r.Get("/settings/palette", s.getPalette)
			r.Get("/settings/groups", s.getGroups)
			r.Get("/baselines", s.listBaselines)
			r.Get("/baselines/{id}", s.getBaseline)
		})

		// Write endpoints: editors only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireEditor)
			r.Put("/settings/public-read", s.setPublicRead)
			r.Put("/settings/palette", s.setPalette)
			r.Put("/settings/groups", s.setGroups)

			r.Post("/swimlanes", s.createSwimlane)
			r.Put("/swimlanes/{id}", s.updateSwimlane)
			r.Delete("/swimlanes/{id}", s.deleteSwimlane)
			r.Post("/swimlanes/{id}/move", s.moveSwimlane)
			r.Post("/swimlanes/{id}/sublanes", s.createSubLane)

			r.Put("/sublanes/{id}", s.updateSubLane)
			r.Delete("/sublanes/{id}", s.deleteSubLane)

			r.Post("/items", s.createItem)
			r.Put("/items/{id}", s.updateItem)
			r.Delete("/items/{id}", s.deleteItem)

			r.Post("/links", s.addLink)
			r.Delete("/links", s.removeLink)

			// baselines (P2)
			r.Post("/baselines", s.createBaseline)
			r.Delete("/baselines/{id}", s.deleteBaseline)
		})
	})

	if s.staticDir != "" {
		s.mountStatic(r)
	}
	return r
}

// ── middleware ──────────────────────────────────────────────────────────────

func (s *Server) requireEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.IsAuthed(r) {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireReadAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth.IsAuthed(r) {
			next.ServeHTTP(w, r)
			return
		}
		ok, err := s.store.GetPublicRead(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusUnauthorized, "login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── auth handlers ───────────────────────────────────────────────────────────

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "atlas"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.auth.Verify(in.Username, in.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := s.auth.SetCookie(w, in.Username); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": s.auth.IsAuthed(r)})
}

// ── plan & settings ─────────────────────────────────────────────────────────

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetPlan(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) getPublicRead(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.store.GetPublicRead(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled})
}

func (s *Server) setPublicRead(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetPublicRead(r.Context(), in.Enabled); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": in.Enabled})
}

func (s *Server) getPalette(w http.ResponseWriter, r *http.Request) {
	colors, err := s.store.GetPalette(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"colors": colors})
}

func (s *Server) setPalette(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Colors []string `json:"colors"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetPalette(r.Context(), in.Colors); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"colors": in.Colors})
}

func (s *Server) getGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.store.GetGroups(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}

func (s *Server) setGroups(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Groups []store.Group `json:"groups"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetGroups(r.Context(), in.Groups); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": in.Groups})
}

// ── swimlanes ───────────────────────────────────────────────────────────────

func (s *Server) createSwimlane(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if in.ID == "" {
		in.ID = uuid.NewString()
	}
	sw, err := s.store.CreateSwimlane(r.Context(), in.ID, in.Name, in.Color)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sw)
}

func (s *Server) updateSwimlane(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.UpdateSwimlane(r.Context(), chi.URLParam(r, "id"), in.Name, in.Color); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteSwimlane(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSwimlane(r.Context(), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) moveSwimlane(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Dir int `json:"dir"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.MoveSwimlane(r.Context(), chi.URLParam(r, "id"), in.Dir); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// ── sub-lanes ───────────────────────────────────────────────────────────────

func (s *Server) createSubLane(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if in.ID == "" {
		in.ID = uuid.NewString()
	}
	sub, err := s.store.CreateSubLane(r.Context(), chi.URLParam(r, "id"), in.ID, in.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (s *Server) updateSubLane(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.UpdateSubLane(r.Context(), chi.URLParam(r, "id"), in.Name); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteSubLane(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSubLane(r.Context(), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// ── items ───────────────────────────────────────────────────────────────────

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	var it store.Item
	if !decode(w, r, &it) {
		return
	}
	if it.SwimlaneID == "" {
		writeErr(w, http.StatusBadRequest, "swimlaneId is required")
		return
	}
	if it.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	if it.ID == "" {
		it.ID = uuid.NewString()
	}
	created, err := s.store.CreateItem(r.Context(), it)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	var it store.Item
	if !decode(w, r, &it) {
		return
	}
	if err := s.store.UpdateItem(r.Context(), chi.URLParam(r, "id"), it); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteItem(r.Context(), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// ── links ───────────────────────────────────────────────────────────────────

func (s *Server) addLink(w http.ResponseWriter, r *http.Request) {
	var in store.Link
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.AddLink(r.Context(), in.A, in.B); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) removeLink(w http.ResponseWriter, r *http.Request) {
	var in store.Link
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.RemoveLink(r.Context(), in.A, in.B); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// ── static SPA ──────────────────────────────────────────────────────────────

// mountStatic serves the built frontend and falls back to index.html so the
// single-page app handles client-side routing. This lets one container ship
// both the API and the UI.
func (s *Server) mountStatic(r chi.Router) {
	fileServer := http.FileServer(http.Dir(s.staticDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		p := filepath.Join(s.staticDir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(s.staticDir, "index.html"))
	})
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrLocked):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
