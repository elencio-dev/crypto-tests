// Package httpapi exposes the engine's algorithm registry over a small REST API:
// list what's available, then run one and get back its full step trace.
package httpapi

import (
	"encoding/json"
	"net/http"

	"crypto-lab/internal/engine"
)

type Server struct {
	registry *engine.Registry
}

func NewServer(r *engine.Registry) *Server {
	return &Server{registry: r}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/algorithms", s.listAlgorithms)
	mux.HandleFunc("POST /api/algorithms/{id}/run", s.runAlgorithm)
	return withCORS(mux)
}

func (s *Server) listAlgorithms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.List())
}

func (s *Server) runAlgorithm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	algo, ok := s.registry.Get(id)
	if !ok {
		http.Error(w, "unknown algorithm: "+id, http.StatusNotFound)
		return
	}
	trace, err := algo.Run()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// withCORS allows the Angular dev server (a different origin) to call this API directly.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
