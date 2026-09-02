// Package server wires up the TTR HTTP API.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// New builds the chi router for the TTR server.
func New() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
