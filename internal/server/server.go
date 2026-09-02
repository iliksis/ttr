// Package server wires up the TTR HTTP API.
package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// Server holds the dependencies TTR's HTTP handlers need.
type Server struct {
	db           *sqlx.DB
	ingestionKey string
}

// New builds the chi router for the TTR server. ingestionKey is the
// Bearer token required on Ingestion requests; an empty key rejects every
// Ingestion request, since there is then no key an Extension could present.
func New(db *sqlx.DB, ingestionKey string) http.Handler {
	s := &Server{db: db, ingestionKey: ingestionKey}

	r := chi.NewRouter()
	r.Get("/health", handleHealth)
	r.With(s.requireIngestionKey).Post("/api/ingest", s.handleIngest)
	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
