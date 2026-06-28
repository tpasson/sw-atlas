// Package api wires the HTTP routes: public auth endpoints, read endpoints
// gated by the public-read switch, and write endpoints restricted to editors.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

		// Public discovery directory for the landing page (no auth).
		r.Get("/explore", s.explore)

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
			r.Get("/settings/ui", s.getUISettings)
			r.Get("/settings/git-colors", s.getGitColors)
			r.Get("/item-types", s.getItemTypes)
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
			r.Put("/settings/ui", s.setUISettings)
			r.Put("/settings/git-colors", s.setGitColors)

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
			r.Post("/items/{id}/scm-refresh", s.scmRefreshItem)

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
			r.Post("/share-scopes/{id}/publish", s.setShareScopePublished)
			r.Post("/share-scopes/{id}/tokens", s.createShareToken)
			r.Get("/share-scopes/{id}/tokens", s.listShareTokens)
			r.Delete("/share-tokens/{id}", s.revokeShareToken)
			// intra-server directory: scopes other users have published
			r.Get("/shares/available", s.listAvailableShares)

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
			// curate the explore page
			r.Put("/workspaces/{slug}/featured", s.setWorkspaceFeatured)
		})
	})

	if s.staticDir != "" {
		s.mountStatic(r)
	}
	return r
}

// ── workspace resolution ──────────────────────────────────────────────────────

// wsContextKey carries the workspace id a request resolved to, set by the access
// middleware so handlers don't re-resolve (or re-authorise) it.
type wsContextKey struct{}

func withWorkspace(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, wsContextKey{}, id)
}

// currentWorkspace returns the workspace a handler operates on: the one resolved
// and authorised by the access middleware (stored in the request context),
// falling back to the session's own workspace and finally the default.
func (s *Server) currentWorkspace(r *http.Request) string {
	if id, ok := r.Context().Value(wsContextKey{}).(string); ok && id != "" {
		return id
	}
	if sess, ok := s.auth.SessionFromRequest(r); ok && sess.WorkspaceID != "" {
		return sess.WorkspaceID
	}
	return store.DefaultWorkspaceID
}

// requestedSlug reads the target workspace slug from the X-Atlas-Workspace header
// (the SPA sets it from the /{slug} URL). Empty means "no explicit target".
func requestedSlug(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(r.Header.Get("X-Atlas-Workspace")))
}

// resolveTargetWorkspace maps the requested slug to a workspace id. With no slug,
// an authenticated request defaults to its own workspace and an anonymous one to
// the default workspace. Returns store.ErrNotFound for an unknown slug.
func (s *Server) resolveTargetWorkspace(r *http.Request) (string, error) {
	slug := requestedSlug(r)
	if slug == "" {
		if sess, ok := s.auth.SessionFromRequest(r); ok && sess.WorkspaceID != "" {
			return sess.WorkspaceID, nil
		}
		return store.DefaultWorkspaceID, nil
	}
	ws, err := s.store.GetWorkspaceBySlug(r.Context(), slug)
	if err != nil {
		return "", err
	}
	return ws.ID, nil
}

// ── middleware ──────────────────────────────────────────────────────────────

// requireEditor restricts writes to the authenticated user's own workspace; a
// request that explicitly targets another workspace is refused.
func (s *Server) requireEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.auth.SessionFromRequest(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		target, err := s.resolveTargetWorkspace(r)
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if target != sess.WorkspaceID {
			writeErr(w, http.StatusForbidden, "you can only edit your own plan")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), sess.WorkspaceID)))
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

// requireReadAccess resolves the target workspace and allows the request when the
// caller owns it or it is public; otherwise 401 (anonymous) or 403 (private).
func (s *Server) requireReadAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, err := s.resolveTargetWorkspace(r)
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		sess, authed := s.auth.SessionFromRequest(r)
		if !(authed && sess.WorkspaceID == target) {
			public, err := s.store.GetPublicRead(r.Context(), target)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			if !public {
				if authed {
					writeErr(w, http.StatusForbidden, "this plan is private")
				} else {
					writeErr(w, http.StatusUnauthorized, "login required")
				}
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
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
	// The workspace slug (for the /{slug} URL) lives on the workspace, not the user.
	slug := u.WorkspaceID
	if ws, err := s.store.GetWorkspace(r.Context(), u.WorkspaceID); err == nil {
		slug = ws.Slug
	}
	sess := auth.Session{UserID: u.ID, Username: u.Username, Role: u.Role, WorkspaceID: u.WorkspaceID, WorkspaceSlug: slug}
	if err := s.auth.SetCookie(w, sess); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "username": u.Username, "role": u.Role, "workspace": slug})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if sess, ok := s.auth.SessionFromRequest(r); ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true, "username": sess.Username, "role": sess.Role, "workspace": sess.WorkspaceSlug,
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

// getUISettings returns the viewed workspace's display settings (so a plan
// renders the way its owner configured it). nil = never set → client defaults.
func (s *Server) getUISettings(w http.ResponseWriter, r *http.Request) {
	raw, err := s.store.GetUISettings(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	var settings any
	if raw != nil {
		settings = raw
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (s *Server) setUISettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Settings json.RawMessage `json:"settings"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetUISettings(r.Context(), s.currentWorkspace(r), in.Settings); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// getGitColors returns the viewed workspace's colour scheme for synced GitHub/
// Gitea items (defaults applied for unset fields).
func (s *Server) getGitColors(w http.ResponseWriter, r *http.Request) {
	c, err := s.store.GetGitColors(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// getItemTypes returns the workspace's item-type catalog (built-ins for now).
func (s *Server) getItemTypes(w http.ResponseWriter, r *http.Request) {
	types, err := s.store.ListItemTypes(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types)
}

func (s *Server) setGitColors(w http.ResponseWriter, r *http.Request) {
	var in store.GitColors
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetGitColors(r.Context(), s.currentWorkspace(r), in); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, in)
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

// scmRefreshItem polls a native item's linked PR/issue/release and reflects its
// live state in the item's progress. Returns the resolved status word.
func (s *Server) scmRefreshItem(w http.ResponseWriter, r *http.Request) {
	status, err := s.store.RefreshItemSCM(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
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
