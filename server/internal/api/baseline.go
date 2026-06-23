package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) listBaselines(w http.ResponseWriter, r *http.Request) {
	bs, err := s.store.ListBaselines(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) getBaseline(w http.ResponseWriter, r *http.Request) {
	b, err := s.store.GetBaseline(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) createBaseline(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
		Note string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	b, err := s.store.CreateBaseline(r.Context(), uuid.NewString(), in.Name, in.Note)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) deleteBaseline(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteBaseline(r.Context(), chi.URLParam(r, "id")); err != nil {
		s.fail(w, err)
		return
	}
	writeNoContent(w)
}
