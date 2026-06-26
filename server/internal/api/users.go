package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/tpasson/sw-atlas/server/internal/auth"
	"github.com/tpasson/sw-atlas/server/internal/store"
)

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
	if username == "" || in.Password == "" {
		writeErr(w, http.StatusBadRequest, "username and password are required")
		return
	}
	if username == store.DefaultWorkspaceID {
		writeErr(w, http.StatusBadRequest, "that username is reserved")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	u, err := s.store.CreateUser(r.Context(), username, hash, in.Role)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// setUserRole changes a user's role (admin/editor).
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

// setUserPassword resets another user's password (admin only).
func (s *Server) setUserPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Password == "" {
		writeErr(w, http.StatusBadRequest, "password is required")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.store.SetUserPassword(r.Context(), chi.URLParam(r, "id"), hash); err != nil {
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
	if in.Password == "" {
		writeErr(w, http.StatusBadRequest, "password is required")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.store.SetUserPassword(r.Context(), sess.UserID, hash); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}
