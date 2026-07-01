package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tpasson/sw-atlas/server/internal/store"
)

// randomToken returns a 256-bit URL-safe bearer secret (shown to the editor once).
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken is what we persist and look up by — the raw token is never stored.
func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// ── scopes (editor) ───────────────────────────────────────────────────────────

func (s *Server) createShareScope(w http.ResponseWriter, r *http.Request) {
	var in store.ShareScope
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	in.ID = uuid.NewString()
	sc, err := s.store.CreateShareScope(r.Context(), s.currentWorkspace(r), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

func (s *Server) listShareScopes(w http.ResponseWriter, r *http.Request) {
	scopes, err := s.store.ListShareScopes(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scopes": scopes})
}

func (s *Server) getShareScope(w http.ResponseWriter, r *http.Request) {
	sc, err := s.store.GetShareScope(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sc)
}

func (s *Server) deleteShareScope(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteShareScope(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// setShareScopePublished toggles a scope's server-wide discoverability (Slice D).
func (s *Server) setShareScopePublished(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Published bool `json:"published"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetShareScopePublished(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Published); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"published": in.Published})
}

// listAvailableShares is the server-wide directory of scopes other users have
// published, so this workspace can subscribe to them locally.
func (s *Server) listAvailableShares(w http.ResponseWriter, r *http.Request) {
	shares, err := s.store.ListPublishedScopes(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shares": shares})
}

// ── tokens (editor) ───────────────────────────────────────────────────────────

func (s *Server) createShareToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Label     string  `json:"label"`
		ExpiresAt *string `json:"expiresAt"`
	}
	if !decode(w, r, &in) {
		return
	}
	var exp *time.Time
	if in.ExpiresAt != nil && *in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "expiresAt must be RFC3339")
			return
		}
		exp = &t
	}
	raw, err := randomToken()
	if err != nil {
		s.internalError(w, err)
		return
	}
	tok, err := s.store.CreateShareToken(r.Context(), s.currentWorkspace(r), uuid.NewString(), chi.URLParam(r, "id"), hashToken(raw), in.Label, exp)
	if err != nil {
		s.fail(w, err)
		return
	}
	// The raw token is returned exactly once — the client builds the subscribe link.
	writeJSON(w, http.StatusCreated, map[string]any{"token": tok, "secret": raw})
}

func (s *Server) listShareTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.store.ListShareTokens(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

func (s *Server) revokeShareToken(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RevokeShareToken(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// ── the subscribe feed (token-authenticated, no session) ──────────────────────

// sharedFeed serves a scope's resolved plan to a subscriber. Auth is the bearer
// token (not a session). It is ETag-aware so a smart-poll consumer that sends
// If-None-Match gets a cheap 304 when nothing changed.
func (s *Server) sharedFeed(w http.ResponseWriter, r *http.Request) {
	raw := bearerToken(r)
	if raw == "" {
		writeErr(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	ws, scopeID, detail, err := s.store.ResolveToken(r.Context(), hashToken(raw))
	if err != nil {
		if err == store.ErrInvalidToken {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		s.fail(w, err)
		return
	}
	plan, err := s.store.ResolveScopePlan(r.Context(), ws, scopeID, detail)
	if err != nil {
		s.fail(w, err)
		return
	}
	env := newEnvelope("share", plan)
	body, err := json.Marshal(env)
	if err != nil {
		s.internalError(w, err)
		return
	}
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
