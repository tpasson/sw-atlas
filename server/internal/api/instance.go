package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tpasson/sw-atlas/server/internal/store"
)

// appVersion is shown in the admin server panel (keep in sync with the frontend).
const appVersion = "1.5.0"

// getInstanceUISettings returns the global (instance-wide) Display config. Readable
// by anyone — every plan renders with it.
func (s *Server) getInstanceUISettings(w http.ResponseWriter, r *http.Request) {
	raw, err := s.store.GetInstanceSetting(r.Context(), "ui_settings")
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": raw})
}

// setInstanceUISettings stores the global Display config (site admin only).
func (s *Server) setInstanceUISettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Settings json.RawMessage `json:"settings"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetInstanceSetting(r.Context(), "ui_settings", in.Settings); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// getServerInfo returns editable server settings + read-only stats (site admin only).
func (s *Server) getServerInfo(w http.ResponseWriter, r *http.Request) {
	raw, err := s.store.GetInstanceSetting(r.Context(), "server")
	if err != nil {
		s.fail(w, err)
		return
	}
	stats, err := s.store.GetInstanceStats(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	settings := raw
	if settings == nil {
		settings = json.RawMessage("{}")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings": settings,
		"stats": map[string]any{
			"version":       appVersion,
			"startedAt":     s.startedAt.Format(time.RFC3339),
			"uptimeSeconds": int(time.Since(s.startedAt).Seconds()),
			"users":         stats.Users,
			"workspaces":    stats.Workspaces,
			"items":         stats.Items,
		},
	})
}

// setServerSettings stores the editable server settings (site admin only).
func (s *Server) setServerSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Settings json.RawMessage `json:"settings"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetInstanceSetting(r.Context(), "server", in.Settings); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}

// getLimits returns the instance quotas / write-rate limit (admin panel).
func (s *Server) getLimits(w http.ResponseWriter, r *http.Request) {
	lim, err := s.store.GetLimits(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lim)
}

// setLimits persists the instance quotas and refreshes the in-memory cache the
// per-user write throttle reads.
func (s *Server) setLimits(w http.ResponseWriter, r *http.Request) {
	var lim store.Limits
	if !decode(w, r, &lim) {
		return
	}
	if err := s.store.SetLimits(r.Context(), lim); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.setLimitsCache(lim)
	writeJSON(w, http.StatusOK, lim)
}
