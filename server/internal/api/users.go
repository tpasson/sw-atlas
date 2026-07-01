package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/tpasson/sw-atlas/server/internal/auth"
	"github.com/tpasson/sw-atlas/server/internal/store"
)

// validPassword enforces a minimum password length; writes a 400 and returns
// false when too short. (Blocks trivially short / empty passwords.)
func validPassword(w http.ResponseWriter, pw string) bool {
	if len(pw) < 12 {
		writeErr(w, http.StatusBadRequest, "password must be at least 12 characters")
		return false
	}
	return true
}

// listUsers returns all accounts (admin only).
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

// createUser provisions a new account with its own personal workspace.
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	username := strings.ToLower(strings.TrimSpace(in.Username))
	if username == "" {
		writeErr(w, http.StatusBadRequest, "username is required")
		return
	}
	if !validPassword(w, in.Password) {
		return
	}
	if username == store.DefaultWorkspaceID {
		writeErr(w, http.StatusBadRequest, "that username is reserved")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	u, err := s.store.CreateUser(r.Context(), username, hash, in.Role)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// setUserRole changes a user's account role (admin/user).
func (s *Server) setUserRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetUserRole(r.Context(), chi.URLParam(r, "id"), in.Role); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// reissueRenamed refreshes the caller's own cookie after a rename (new username,
// new home slug, bumped token version) so they stay logged in.
func (s *Server) reissueRenamed(w http.ResponseWriter, r *http.Request, sess auth.Session, newName string, newTV int) bool {
	sess.Username = newName
	sess.WorkspaceSlug = newName // home slug now follows the username
	sess.TokenVersion = newTV
	if err := s.auth.SetCookie(w, r, sess); err != nil {
		s.internalError(w, err)
		return false
	}
	return true
}

// renameUser changes any account's login username + personal-workspace URL (admin
// only). If the admin renamed their own account, their cookie is re-issued.
func (s *Server) renameUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
	}
	if !decode(w, r, &in) {
		return
	}
	id := chi.URLParam(r, "id")
	newSlug, newTV, err := s.store.RenameUser(r.Context(), id, in.Username)
	if err != nil {
		s.fail(w, err)
		return
	}
	if sess, ok := s.auth.SessionFromRequest(r); ok && sess.UserID == id {
		if !s.reissueRenamed(w, r, sess, newSlug, newTV) {
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": newSlug, "slug": newSlug})
}

// renameOwnUsername lets any signed-in user change their own username + personal
// workspace URL (self-service). The name must be free and not reserved.
func (s *Server) renameOwnUsername(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authedSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var in struct {
		Username string `json:"username"`
	}
	if !decode(w, r, &in) {
		return
	}
	newSlug, newTV, err := s.store.RenameUser(r.Context(), sess.UserID, in.Username)
	if err != nil {
		s.fail(w, err)
		return
	}
	if !s.reissueRenamed(w, r, sess, newSlug, newTV) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": newSlug, "slug": newSlug})
}

// updateOwnProfile lets any signed-in user set their real name + email (all
// optional). A non-empty email must contain "@".
func (s *Server) updateOwnProfile(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.authedSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var in struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	if e := strings.TrimSpace(in.Email); e != "" && !strings.Contains(e, "@") {
		writeErr(w, http.StatusBadRequest, "that doesn't look like an email address")
		return
	}
	if err := s.store.UpdateUserProfile(r.Context(), sess.UserID, in.FirstName, in.LastName, in.Email); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// setUserPassword resets another user's password (admin only).
func (s *Server) setUserPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validPassword(w, in.Password) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if _, err := s.store.SetUserPassword(r.Context(), chi.URLParam(r, "id"), hash); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// deleteUser removes an account and its workspace (admin only). An admin cannot
// delete their own account, to avoid logging themselves out mid-session.
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if sess, ok := s.auth.SessionFromRequest(r); ok && sess.UserID == id {
		writeErr(w, http.StatusConflict, "you cannot delete your own account")
		return
	}
	if err := s.store.DeleteUser(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// changeOwnPassword lets the logged-in user set their own password.
func (s *Server) changeOwnPassword(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.auth.SessionFromRequest(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validPassword(w, in.Password) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.internalError(w, err)
		return
	}
	newTV, err := s.store.SetUserPassword(r.Context(), sess.UserID, hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	// Re-issue the caller's own cookie with the new token version so they stay
	// logged in, while every OTHER existing session for this account is revoked.
	sess.TokenVersion = newTV
	if err := s.auth.SetCookie(w, r, sess); err != nil {
		s.internalError(w, err)
		return
	}
	writeNoContent(w)
}
