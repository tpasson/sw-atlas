package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tpasson/sw-atlas/server/internal/store"
)

// createSubscription registers a subscription from either a subscribe code
// (base64 of {u,t,n}) or an explicit {url, token, label}, then does a first sync.
func (s *Server) createSubscription(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code            string `json:"code"`
		URL             string `json:"url"`
		Token           string `json:"token"`
		Label           string `json:"label"`
		IntervalSeconds int    `json:"intervalSeconds"`
	}
	if !decode(w, r, &in) {
		return
	}

	url, token, label := in.URL, in.Token, in.Label
	if in.Code != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.Code))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid subscribe code")
			return
		}
		var c struct {
			U string `json:"u"`
			T string `json:"t"`
			N string `json:"n"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid subscribe code")
			return
		}
		url, token = c.U, c.T
		if label == "" {
			label = c.N
		}
	}
	if url == "" || token == "" {
		writeErr(w, http.StatusBadRequest, "url and token are required")
		return
	}
	if label == "" {
		label = "Subscription"
	}

	ws := s.currentWorkspace(r)
	sub, err := s.store.CreateSubscription(r.Context(), ws, uuid.NewString(), label, url, token, in.IntervalSeconds)
	if err != nil {
		s.fail(w, err)
		return
	}
	// First sync immediately; failures surface in the subscription's last_status.
	_ = s.store.SyncSubscription(r.Context(), ws, sub.ID)
	out, err := s.store.GetSubscription(r.Context(), ws, sub.ID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := s.store.ListSubscriptions(r.Context(), s.currentWorkspace(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": subs})
}

func (s *Server) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSubscription(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) syncSubscription(w http.ResponseWriter, r *http.Request) {
	ws := s.currentWorkspace(r)
	id := chi.URLParam(r, "id")
	if err := s.store.SyncSubscription(r.Context(), ws, id); err != nil {
		if err == store.ErrNotFound {
			s.fail(w, err)
			return
		}
		// A sync error (e.g. remote unreachable) is recorded in last_status; still
		// return the subscription so the client can show what happened.
	}
	out, err := s.store.GetSubscription(r.Context(), ws, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) setSubscriptionPaused(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Paused bool `json:"paused"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetSubscriptionPaused(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Paused); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

func (s *Server) setSwimlaneHidden(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Hidden bool `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetSwimlaneHidden(r.Context(), s.currentWorkspace(r), chi.URLParam(r, "id"), in.Hidden); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}
