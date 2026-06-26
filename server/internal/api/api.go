// Package api wires the HTTP routes: public auth endpoints, read endpoints
// gated by the public-read switch, and write endpoints restricted to editors.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

		// Subscribe feed: authenticated by its bearer token, not a session.
		r.Get("/shared", s.sharedFeed)

		// Read endpoints: open when public-read is on, otherwise editor-only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireReadAccess)
			r.Get("/plan", s.getPlan)
			r.Get("/export", s.exportPlan)
			r.Get("/settings/public-read", s.getPublicRead)
			r.Get("/settings/palette", s.getPalette)
			r.Get("/settings/groups", s.getGroups)
			r.Get("/baselines", s.listBaselines)
			r.Get("/baselines/{id}", s.getBaseline)
		})

		// Write endpoints: editors only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireEditor)
			r.Post("/import", s.importPlan)
			r.Put("/settings/public-read", s.setPublicRead)
			r.Put("/settings/palette", s.setPalette)
			r.Put("/settings/groups", s.setGroups)

			r.Post("/swimlanes", s.createSwimlane)
			r.Post("/swimlanes/reorder", s.reorderSwimlanes)
			r.Put("/swimlanes/{id}", s.updateSwimlane)
			r.Delete("/swimlanes/{id}", s.deleteSwimlane)
			r.Post("/swimlanes/{id}/move", s.moveSwimlane)
			r.Post("/swimlanes/{id}/sublanes", s.createSubLane)

			r.Post("/sublanes/reorder", s.reorderSubLanes)
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

			// share scopes & subscribe tokens (federation, producer side)
			r.Post("/share-scopes", s.createShareScope)
			r.Get("/share-scopes", s.listShareScopes)
			r.Get("/share-scopes/{id}", s.getShareScope)
			r.Delete("/share-scopes/{id}", s.deleteShareScope)
			r.Post("/share-scopes/{id}/tokens", s.createShareToken)
			r.Get("/share-scopes/{id}/tokens", s.listShareTokens)
			r.Delete("/share-tokens/{id}", s.revokeShareToken)

			// subscriptions (federation, consumer side)
			r.Post("/subscriptions", s.createSubscription)
			r.Get("/subscriptions", s.listSubscriptions)
			r.Delete("/subscriptions/{id}", s.deleteSubscription)
			r.Post("/subscriptions/{id}/sync", s.syncSubscription)
			r.Post("/subscriptions/{id}/pause", s.setSubscriptionPaused)
			r.Post("/swimlanes/{id}/hidden", s.setSwimlaneHidden)

			// GitHub sources (releases/tags/issues/PRs → read-only swimlane)
			r.Post("/github-sources", s.createGitHubSource)
			r.Get("/github-sources", s.listGitHubSources)
			r.Post("/github-sources/{id}/sync", s.syncGitHubSource)
			r.Post("/github-sources/{id}/token", s.setGitHubSourceToken)
			r.Delete("/github-sources/{id}", s.deleteGitHubSource)

			// self-service: change your own password
			r.Put("/account/password", s.changeOwnPassword)
		})

		// User administration: admins only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdmin)
			r.Get("/users", s.listUsers)
			r.Post("/users", s.createUser)
			r.Put("/users/{id}/role", s.setUserRole)
			r.Put("/users/{id}/password", s.setUserPassword)
			r.Delete("/users/{id}", s.deleteUser)
		})
	})

	if s.staticDir != "" {
		s.mountStatic(r)
	}
	return r
}

// ── workspace ───────────────────────────────────────────────────────────────

// currentWorkspace resolves the workspace a request operates on. An authenticated
// request operates on the session user's personal workspace; anonymous (public)
// requests fall back to the default workspace. Slice C will additionally derive
// it from the URL (/{username}).
func (s *Server) currentWorkspace(r *http.Request) string {
	if sess, ok := s.auth.SessionFromRequest(r); ok && sess.WorkspaceID != "" {
		return sess.WorkspaceID
	}
	return store.DefaultWorkspaceID
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

// requireAdmin gates account-management endpoints to admins only.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.auth.SessionFromRequest(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if sess.Role != store.RoleAdmin {
			writeErr(w, http.StatusForbidden, "admin access required")
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
		ok, err := s.store.GetPublicRead(r.Context(), s.currentWorkspace(r))
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
	u, hash, err := s.store.GetUserByUsername(r.Context(), in.Username)
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	sess := auth.Session{UserID: u.ID, Username: u.Username, Role: u.Role, WorkspaceID: u.WorkspaceID}
	if err := s.auth.SetCookie(w, sess); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "username": u.Username, "role": u.Role})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if sess, ok := s.auth.SessionFromRequest(r); ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true, "username": sess.Username, "role": sess.Role,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

// ── plan & settings ─────────────────────────────────────────────────────────

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetPlan(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// schemaVersion is the version of the ATLAS wire format (export / import / share
// all share this envelope). Bump only on breaking changes.
const schemaVersion = 1

// planEnvelope is the portable wire format: a versioned header plus the plan.
// The same shape powers file export/import today and the live-share feed later.
type planEnvelope struct {
	Atlas struct {
		Schema      int    `json:"schema"`
		Kind        string `json:"kind"` // "export" | "share"
		GeneratedAt string `json:"generatedAt,omitempty"`
	} `json:"atlas"`
	Swimlanes  []store.Swimlane `json:"swimlanes"`
	Milestones []store.Item     `json:"milestones"`
	Links      []store.Link     `json:"links"`
}

func newEnvelope(kind string, p store.Plan) planEnvelope {
	var env planEnvelope
	env.Atlas.Schema = schemaVersion
	env.Atlas.Kind = kind
	env.Atlas.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	env.Swimlanes, env.Milestones, env.Links = p.Swimlanes, p.Milestones, p.Links
	return env
}

// exportPlan returns the whole plan as a portable JSON envelope (backup / move /
// hand to a colleague). Gated like /plan (read access).
func (s *Server) exportPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetPlan(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newEnvelope("export", p))
}

// importPlan additively imports an envelope into the current plan (editor-only,
// Copy-mode: new IDs, provenance stripped). Returns counts of what was created.
func (s *Server) importPlan(w http.ResponseWriter, r *http.Request) {
	var env planEnvelope
	if !decode(w, r, &env) {
		return
	}
	if env.Atlas.Schema > schemaVersion {
		writeErr(w, http.StatusBadRequest, "export was made with a newer ATLAS version")
		return
	}
	sum, err := s.store.ImportPlan(r.Context(), s.currentWorkspace(r), store.Plan{
		Swimlanes:  env.Swimlanes,
		Milestones: env.Milestones,
		Links:      env.Links,
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) getPublicRead(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.store.GetPublicRead(r.Context(), s.currentWorkspace(r))
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
	if err := s.store.SetPublicRead(r.Context(), s.currentWorkspace(r), in.Enabled); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": in.Enabled})
}

func (s *Server) getPalette(w http.ResponseWriter, r *http.Request) {
	colors, err := s.store.GetPalette(r.Context(), s.currentWorkspace(r))
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
	if err := s.store.SetPalette(r.Context(), s.currentWorkspace(r), in.Colors); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"colors": in.Colors})
}

func (s *Server) getGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.store.GetGroups(r.Context(), s.currentWorkspace(r))
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
	if err := s.store.SetGroups(r.Context(), s.currentWorkspace(r), in.Groups); err != nil {
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
	sw, err := s.store.CreateSwimlane(r.Context(), s.currentWorkspace(r), in.ID, in.Name, in.Color)
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
	if err := s.store.UpdateSwimlane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Name, in.Color); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteSwimlane(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSwimlane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) reorderSwimlanes(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []string `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.ReorderSwimlanes(r.Context(), s.currentWorkspace(r), in.IDs); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) reorderSubLanes(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs []string `json:"ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.ReorderSubLanes(r.Context(), s.currentWorkspace(r), in.IDs); err != nil {
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
	if err := s.store.MoveSwimlane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Dir); err != nil {
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
	sub, err := s.store.CreateSubLane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.ID, in.Name)
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
	if err := s.store.UpdateSubLane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Name); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteSubLane(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSubLane(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
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
	created, err := s.store.CreateItem(r.Context(), s.currentWorkspace(r), it)
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
	if err := s.store.UpdateItem(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), it); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteItem(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
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
	if err := s.store.AddLink(r.Context(), s.currentWorkspace(r), in.A, in.B); err != nil {
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
	if err := s.store.RemoveLink(r.Context(), s.currentWorkspace(r), in.A, in.B); err != nil {
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
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrLastAdmin), errors.Is(err, store.ErrProtected):
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
