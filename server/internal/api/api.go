// Package api wires the HTTP routes: public auth endpoints, read endpoints
// gated by the public-read switch, and write endpoints restricted to editors.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	startedAt time.Time

	writeRL *rateLimiter // per-user write throttle (limit is configurable)
	limMu   sync.RWMutex // guards the cached limits below
	lim     store.Limits // cached instance limits (refreshed on PUT /instance/limits)
}

// rateLimiter is a tiny in-memory sliding-window limiter (per client IP), used to
// throttle the login endpoint against brute-force / password-spraying.
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, max: max, window: window}
}

// setMax updates the ceiling (so an admin-configured write limit takes effect).
func (l *rateLimiter) setMax(n int) {
	l.mu.Lock()
	l.max = n
	l.mu.Unlock()
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-l.window)
	fresh := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= l.max {
		l.hits[key] = fresh
		return false
	}
	l.hits[key] = append(fresh, time.Now())
	return true
}

func (l *rateLimiter) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = h
		}
		if !l.allow(ip) {
			writeErr(w, http.StatusTooManyRequests, "too many attempts — please wait a moment")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loginRL: at most 10 login attempts per minute per IP.
var loginRL = newRateLimiter(10, time.Minute)

func (s *Server) currentLimits() store.Limits {
	s.limMu.RLock()
	defer s.limMu.RUnlock()
	return s.lim
}

func (s *Server) setLimitsCache(lim store.Limits) {
	s.limMu.Lock()
	s.lim = lim
	s.limMu.Unlock()
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// writeLimit throttles mutating requests per authenticated user, using the
// admin-configured writesPerMinute (0 = off). Reads/anonymous requests pass
// through untouched. Keying uses the signed cookie only (no DB round-trip).
func (s *Server) writeLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isMutating(r.Method) {
			if max := s.currentLimits().WritesPerMinute; max > 0 {
				if sess, ok := s.auth.SessionFromRequest(r); ok && sess.UserID != "" {
					s.writeRL.setMax(max)
					if !s.writeRL.allow(sess.UserID) {
						writeErr(w, http.StatusTooManyRequests, "you're making changes too quickly — please slow down a moment")
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// maxBodyBytes caps the request body to bound server memory (DoS defense). 16 MB
// is generous for JSON plan imports while still rejecting abusive payloads.
func maxBodyBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func NewRouter(st *store.Store, au *auth.Auth, staticDir string) http.Handler {
	s := &Server{store: st, auth: au, staticDir: staticDir, startedAt: time.Now()}
	s.writeRL = newRateLimiter(store.DefaultLimits.WritesPerMinute, time.Minute)
	if lim, err := st.GetLimits(context.Background()); err == nil {
		s.lim = lim
	} else {
		s.lim = store.DefaultLimits
	}

	// Minimal container images (alpine) ship no /etc/mime.types, so Go can't map
	// .svg → image/svg+xml and the favicon is served as text/xml (browsers then
	// ignore it). Register the type explicitly.
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, maxBodyBytes(16<<20), s.writeLimit)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.health)
		r.With(loginRL.limit).Post("/login", s.login)
		r.Post("/logout", s.logout)
		r.Get("/me", s.me)
		r.Get("/instance/ui-settings", s.getInstanceUISettings) // global Display config (public read)

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
			r.Get("/settings/public-cr", s.getPublicCR)
			r.Get("/settings/palette", s.getPalette)
			r.Get("/settings/groups", s.getGroups)
			r.Get("/settings/ui", s.getUISettings)
			r.Get("/settings/git-colors", s.getGitColors)
			r.Get("/item-types", s.getItemTypes)
			r.Get("/workflows", s.getWorkflows)
			r.Get("/members", s.listWorkspaceMembers)
			r.Get("/baselines", s.listBaselines)
			r.Get("/baselines/{id}", s.getBaseline)
			r.Get("/items/{id}/revisions", s.listItemRevisions)
			r.Get("/items/{id}/revisions/{version}", s.getItemRevision)
		})

		// Projects: any authenticated user lists their own + creates new ones,
		// and can leave a project they belong to.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/projects", s.listProjects)
			r.Post("/projects", s.createProject)
			r.Post("/projects/{slug}/leave", s.leaveProject)
			r.Put("/account/username", s.renameOwnUsername) // self-service rename
			r.Put("/account/password", s.changeOwnPassword) // self-service password
			r.Put("/account/profile", s.updateOwnProfile)   // self-service name + email
		})

		// Member management: only the project owner (by path slug).
		r.Group(func(r chi.Router) {
			r.Use(s.requireWorkspaceOwnerByPath)
			r.Put("/projects/{slug}", s.renameProject)
			r.Delete("/projects/{slug}", s.deleteProject)
			r.Get("/projects/{slug}/members", s.listMembers)
			r.Post("/projects/{slug}/members", s.inviteMember)
			r.Put("/projects/{slug}/members/{userId}/role", s.setMemberRole)
			r.Delete("/projects/{slug}/members/{userId}", s.removeMember)
		})

		// Workspace CONFIGURATION: the owner only (the "blueprint" — types, look,
		// sources, sharing). Editors change content, not configuration.
		r.Group(func(r chi.Router) {
			r.Use(s.requireWorkspaceOwner)
			r.Put("/settings/public-read", s.setPublicRead)
			r.Put("/settings/public-cr", s.setPublicCR)
			r.Put("/settings/palette", s.setPalette)
			r.Put("/settings/ui", s.setUISettings)
			r.Put("/settings/git-colors", s.setGitColors)
			r.Put("/item-types", s.setItemTypes)
			r.Put("/workflows", s.setWorkflows)

			// GitHub sources (releases/tags/issues/PRs → read-only swimlane)
			r.Post("/github-sources", s.createGitHubSource)
			r.Get("/github-sources", s.listGitHubSources)
			r.Post("/github-sources/{id}/sync", s.syncGitHubSource)
			r.Post("/github-sources/{id}/token", s.setGitHubSourceToken)
			r.Delete("/github-sources/{id}", s.deleteGitHubSource)

			// change-request decisions: owner approves/rejects
			r.Post("/change-requests/{id}/approve", s.approveChangeRequest)
			r.Post("/change-requests/{id}/reject", s.rejectChangeRequest)
		})

		// Reviewing (listing) change requests stays member-only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireMember)
			r.Get("/change-requests", s.listChangeRequests)
		})

		// Proposing a change: any member — or, when the project opts in, anyone
		// (account-less) via the public-CR switch.
		r.Group(func(r chi.Router) {
			r.Use(s.requireProposeAccess)
			r.Post("/change-requests", s.createChangeRequest)
		})

		// Write endpoints: editors only (content, not configuration).
		r.Group(func(r chi.Router) {
			r.Use(s.requireEditor)
			r.Post("/import", s.importPlan)
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
		})

		// User administration: admins only.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdmin)
			// Instance/admin config: global Display + server settings/stats.
			r.Put("/instance/ui-settings", s.setInstanceUISettings)
			r.Get("/instance/server", s.getServerInfo)
			r.Put("/instance/server", s.setServerSettings)
			r.Get("/instance/limits", s.getLimits)
			r.Put("/instance/limits", s.setLimits)
			r.Get("/users", s.listUsers)
			r.Post("/users", s.createUser)
			r.Put("/users/{id}/role", s.setUserRole)
			r.Put("/users/{id}/username", s.renameUser)
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

// authedSession parses the session cookie AND confirms it hasn't been revoked:
// the user still exists and the token_version stamped in the JWT matches the
// current one (bumped on password change, role change, or account deletion). Used
// at every gate so a stale/forged-role token can't keep access.
func (s *Server) authedSession(r *http.Request) (auth.Session, bool) {
	sess, ok := s.auth.SessionFromRequest(r)
	if !ok {
		return auth.Session{}, false
	}
	cur, err := s.store.UserTokenVersion(r.Context(), sess.UserID)
	if err != nil || cur != sess.TokenVersion {
		return auth.Session{}, false
	}
	return sess, true
}

// requireEditor allows writes when the caller is an owner or editor MEMBER of the
// targeted workspace (membership is the authorization source).
func (s *Server) requireEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.authedSession(r)
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
			s.internalError(w, err)
			return
		}
		role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, target)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if sess.Role != store.RoleAdmin && role != store.WSRoleOwner && role != store.WSRoleEditor {
			writeErr(w, http.StatusForbidden, "you don't have edit access to this plan")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
	})
}

// requireWorkspaceOwner gates workspace-CONFIGURATION endpoints (types, display,
// palette, sources, sharing, change-request decisions) to the OWNER of the active
// workspace. Editors can change content; only owners change the "blueprint".
func (s *Server) requireWorkspaceOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.authedSession(r)
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
			s.internalError(w, err)
			return
		}
		role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, target)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if sess.Role != store.RoleAdmin && role != store.WSRoleOwner {
			writeErr(w, http.StatusForbidden, "only the project owner can change this")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
	})
}

// requireMember gates an endpoint to any authenticated MEMBER of the active
// workspace (viewer / editor / owner) — used for proposing change requests.
func (s *Server) requireMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.authedSession(r)
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
			s.internalError(w, err)
			return
		}
		role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, target)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if role == "" && sess.Role != store.RoleAdmin {
			writeErr(w, http.StatusForbidden, "you are not a member of this project")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
	})
}

// requireProposeAccess gates change-request creation: any member (or site admin)
// may propose; when the project opts in via the public-CR switch, anyone may —
// even without an account.
func (s *Server) requireProposeAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, err := s.resolveTargetWorkspace(r)
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			s.internalError(w, err)
			return
		}
		if sess, authed := s.authedSession(r); authed {
			role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, target)
			if err != nil {
				s.internalError(w, err)
				return
			}
			if role != "" || sess.Role == store.RoleAdmin {
				next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
				return
			}
		}
		pub, err := s.store.GetPublicCR(r.Context(), target)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if !pub {
			writeErr(w, http.StatusForbidden, "change requests aren't open to the public for this project")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), target)))
	})
}

// requireAuth gates an endpoint to any authenticated user (no workspace target).
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.authedSession(r); !ok {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireWorkspaceOwnerByPath gates member-management endpoints: the caller must
// be an OWNER of the workspace named by the {slug} path param (which may differ
// from the active workspace). Stores the resolved id in context for the handler.
func (s *Server) requireWorkspaceOwnerByPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.authedSession(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ws, err := s.store.GetWorkspaceBySlug(r.Context(), chi.URLParam(r, "slug"))
		if err == store.ErrNotFound {
			writeErr(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			s.internalError(w, err)
			return
		}
		role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, ws.ID)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if sess.Role != store.RoleAdmin && role != store.WSRoleOwner {
			writeErr(w, http.StatusForbidden, "only the project owner can manage members")
			return
		}
		next.ServeHTTP(w, r.WithContext(withWorkspace(r.Context(), ws.ID)))
	})
}

// requireAdmin gates account-management endpoints to admins only.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.authedSession(r)
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
			s.internalError(w, err)
			return
		}
		sess, authed := s.authedSession(r)
		allowed := false
		if authed {
			role, err := s.store.RoleInWorkspace(r.Context(), sess.UserID, target)
			if err != nil {
				s.internalError(w, err)
				return
			}
			allowed = role != "" // any member reads, even a private workspace
			if sess.Role == store.RoleAdmin {
				allowed = true // site admins can read every workspace's plan
			}
		}
		if !allowed {
			public, err := s.store.GetPublicRead(r.Context(), target)
			if err != nil {
				s.internalError(w, err)
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
	if err != nil {
		auth.DummyPasswordCheck(in.Password) // equalize timing for unknown usernames
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(hash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	// The workspace slug (for the /{slug} URL) lives on the workspace, not the user.
	slug := u.WorkspaceID
	if ws, err := s.store.GetWorkspace(r.Context(), u.WorkspaceID); err == nil {
		slug = ws.Slug
	}
	sess := auth.Session{UserID: u.ID, Username: u.Username, Role: u.Role, WorkspaceID: u.WorkspaceID, WorkspaceSlug: slug, TokenVersion: u.TokenVersion}
	if err := s.auth.SetCookie(w, r, sess); err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "username": u.Username, "role": u.Role, "workspace": slug})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if sess, ok := s.authedSession(r); ok {
		out := map[string]any{
			"authenticated": true, "username": sess.Username, "role": sess.Role, "workspace": sess.WorkspaceSlug,
		}
		if u, err := s.store.GetUserByID(r.Context(), sess.UserID); err == nil {
			out["email"] = u.Email
			out["firstName"] = u.FirstName
			out["lastName"] = u.LastName
		}
		writeJSON(w, http.StatusOK, out)
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

func (s *Server) getPublicCR(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.store.GetPublicCR(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": enabled})
}

func (s *Server) setPublicCR(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetPublicCR(r.Context(), s.currentWorkspace(r), in.Enabled); err != nil {
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

// listProjects returns the workspaces for the switcher: the caller's own, or —
// for a site admin — every workspace (so admins can see all projects and plans).
func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.auth.SessionFromRequest(r)
	var list []store.UserWorkspace
	var err error
	if sess.Role == store.RoleAdmin {
		list, err = s.store.ListAllWorkspaces(r.Context(), sess.UserID)
	} else {
		list, err = s.store.ListWorkspacesForUser(r.Context(), sess.UserID)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// createProject creates a new workspace owned by the caller.
func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.auth.SessionFromRequest(r)
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	// Instance quota: cap the collaborative projects a user can own.
	if lim := s.currentLimits(); lim.MaxProjectsPerUser > 0 {
		if n, err := s.store.CountOwnedProjects(r.Context(), sess.UserID); err == nil && n >= lim.MaxProjectsPerUser {
			writeErr(w, http.StatusTooManyRequests, "project limit reached — ask an admin to raise it")
			return
		}
	}
	ws, err := s.store.CreateWorkspace(r.Context(), sess.UserID, in.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ws)
}

// renameProject renames a project (owner only).
func (s *Server) renameProject(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.RenameWorkspace(r.Context(), s.currentWorkspace(r), in.Name); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// deleteProject deletes a project and all its data (owner only; not a home ws).
func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteWorkspace(r.Context(), s.currentWorkspace(r)); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// listWorkspaceMembers returns the current workspace's roster for any reader —
// used by the assignee picker and to render assignee avatars.
func (s *Server) listWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListMembers(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	// Email addresses are only exposed to authenticated requesters.
	if _, ok := s.authedSession(r); !ok {
		for i := range list {
			list[i].Email = ""
		}
	}
	writeJSON(w, http.StatusOK, list)
}

// listMembers returns the roster of the workspace named by {slug} (owner only).
func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListMembers(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// inviteMember adds a user (by username) as editor/viewer (owner only).
func (s *Server) inviteMember(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	m, err := s.store.AddMember(r.Context(), s.currentWorkspace(r), in.Username, in.Role)
	if err == store.ErrNotFound {
		writeErr(w, http.StatusNotFound, "no user with that username")
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// setMemberRole changes a member's role (owner only; last-owner protected).
func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetMemberRole(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "userId"), in.Role); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// removeMember removes a member (owner only; last-owner protected).
func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RemoveMember(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "userId")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// leaveProject removes the caller from a project (any member; last owner can't).
func (s *Server) leaveProject(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.auth.SessionFromRequest(r)
	ws, err := s.store.GetWorkspaceBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err == store.ErrNotFound {
		writeErr(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.store.RemoveMember(r.Context(), ws.ID, sess.UserID); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// getItemTypes returns the workspace's item-type catalog (built-ins + custom).
func (s *Server) getItemTypes(w http.ResponseWriter, r *http.Request) {
	types, err := s.store.ListItemTypes(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types)
}

// setItemTypes persists the workspace's custom item types (built-ins are ignored).
func (s *Server) setItemTypes(w http.ResponseWriter, r *http.Request) {
	var in []store.ItemType
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetItemTypes(r.Context(), s.currentWorkspace(r), in); err != nil {
		s.fail(w, err)
		return
	}
	types, err := s.store.ListItemTypes(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types)
}

// getWorkflows returns the workspace's shared, reusable status workflows.
func (s *Server) getWorkflows(w http.ResponseWriter, r *http.Request) {
	wfs, err := s.store.GetWorkflows(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wfs)
}

// setWorkflows replaces the workspace's shared workflows; item types reference
// them by key, so editing one here updates every type that uses it.
func (s *Server) setWorkflows(w http.ResponseWriter, r *http.Request) {
	var in []store.Workflow
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetWorkflows(r.Context(), s.currentWorkspace(r), in); err != nil {
		s.fail(w, err)
		return
	}
	wfs, err := s.store.GetWorkflows(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wfs)
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
	// swimlaneId is optional: off-timeline artifacts (work-item / container types)
	// have no lane and live only in the Explorer.
	if it.Title == "" {
		writeErr(w, http.StatusBadRequest, "title is required")
		return
	}
	if it.ID == "" {
		it.ID = uuid.NewString()
	}
	sess, _ := s.auth.SessionFromRequest(r)
	created, err := s.store.CreateItemAs(r.Context(), s.currentWorkspace(r), sess.UserID, it)
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
	sess, _ := s.auth.SessionFromRequest(r)
	if err := s.store.UpdateItemAs(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), sess.UserID, it); err != nil {
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

// createChangeRequest stores a member's proposed change (pending the owner).
func (s *Server) createChangeRequest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind         string          `json:"kind"`
		TargetItemID string          `json:"targetItemId"`
		Payload      json.RawMessage `json:"payload"`
		Note         string          `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Payload) == 0 {
		writeErr(w, http.StatusBadRequest, "a proposed change is required")
		return
	}
	sess, _ := s.auth.SessionFromRequest(r)
	cr, err := s.store.CreateChangeRequest(r.Context(), s.currentWorkspace(r), uuid.NewString(), sess.UserID, in.Kind, in.TargetItemID, in.Payload, in.Note)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cr)
}

// listChangeRequests returns the workspace's change requests (pending first).
func (s *Server) listChangeRequests(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListChangeRequests(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// approveChangeRequest applies a proposal to the live plan (owner only).
func (s *Server) approveChangeRequest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Note string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, _ := s.auth.SessionFromRequest(r)
	cr, err := s.store.ApproveChangeRequest(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), sess.UserID, in.Note)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cr)
}

// rejectChangeRequest declines a proposal (owner only).
func (s *Server) rejectChangeRequest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Note string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, _ := s.auth.SessionFromRequest(r)
	cr, err := s.store.RejectChangeRequest(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), sess.UserID, in.Note)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cr)
}

// listItemRevisions returns an item's version history (newest first, no snapshots).
func (s *Server) listItemRevisions(w http.ResponseWriter, r *http.Request) {
	revs, err := s.store.ListItemRevisions(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, revs)
}

// getItemRevision returns one version's full snapshot of an item.
func (s *Server) getItemRevision(w http.ResponseWriter, r *http.Request) {
	v, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid version")
		return
	}
	rev, err := s.store.GetItemRevision(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), v)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rev)
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
	if err := s.store.AddLink(r.Context(), s.currentWorkspace(r), in.A, in.B, in.Rel, in.Version); err != nil {
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
	if err := s.store.RemoveLink(r.Context(), s.currentWorkspace(r), in.A, in.B, in.Rel); err != nil {
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
	serve := func(w http.ResponseWriter, req *http.Request) {
		p := filepath.Join(s.staticDir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			// Static asset (favicon, JS, CSS, images): serve as-is. The CSP /
			// nosniff / referrer headers belong on the HTML document — putting them
			// on a favicon is inert, and Safari refuses to render a favicon that
			// carries X-Content-Type-Options: nosniff (or a CSP).
			fileServer.ServeHTTP(w, req)
			return
		}
		// SPA document: this is where the CSP actually applies (it governs what the
		// page may load), so set the security headers only here.
		setSecurityHeaders(w)
		http.ServeFile(w, req, filepath.Join(s.staticDir, "index.html"))
	}
	// Serve both GET and HEAD: a GET-only route answers HEAD with 405, and some
	// reverse proxies / CDNs HEAD static assets (e.g. the favicon) to validate or
	// cache them — a 405 there stops the favicon from being delivered.
	r.Get("/*", serve)
	r.Head("/*", serve)
}

// setSecurityHeaders locks the SPA to its own origin: no external scripts,
// styles, fonts, images or network connections are ever loaded (everything is
// served by this container). 'unsafe-inline' is granted for styles only, because
// Vue uses inline :style bindings — styles can't execute code, so it's harmless.
func setSecurityHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data:; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"object-src 'none'; "+
			"base-uri 'self'; "+
			"frame-ancestors 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
}

// ── helpers ─────────────────────────────────────────────────────────────────

// internalError logs the real error server-side and returns a generic 500 to the
// client, so DB messages / internal paths aren't disclosed to callers.
func (s *Server) internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeErr(w, http.StatusInternalServerError, "internal server error")
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrLocked):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrLastAdmin), errors.Is(err, store.ErrProtected), errors.Is(err, store.ErrLastOwner):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrLimitReached):
		writeErr(w, http.StatusTooManyRequests, err.Error())
	default:
		s.internalError(w, err)
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
