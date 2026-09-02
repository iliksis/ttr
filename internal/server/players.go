package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type rosterEntry struct {
	NUID string `json:"nuid" db:"nuid"`
}

// handleRoster returns every Player's nuid, omitting Players with none yet.
// It's the minimal shape the Extension needs to drive Capture.
func (s *Server) handleRoster(w http.ResponseWriter, r *http.Request) {
	var roster []rosterEntry
	err := s.db.Select(&roster, `
		SELECT nuid FROM players WHERE nuid IS NOT NULL ORDER BY nuid
	`)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if roster == nil {
		roster = []rosterEntry{}
	}

	writeJSON(w, http.StatusOK, roster)
}

type playerSummary struct {
	ID           int64          `json:"id" db:"id"`
	NUID         *string        `json:"nuid" db:"nuid"`
	FirstName    string         `json:"first_name" db:"first_name"`
	LastName     string         `json:"last_name" db:"last_name"`
	LatestTTR    *int           `json:"latest_ttr" db:"latest_ttr"`
	LatestTTRAt  *string        `json:"latest_ttr_at" db:"latest_ttr_at"`
	LatestQTTR   *int           `json:"latest_qttr" db:"latest_qttr"`
	LatestQTTRAt *string        `json:"latest_qttr_at" db:"latest_qttr_at"`
}

// handlePlayers returns every Player along with their most recent Rating
// snapshot of each type, null where none has been captured yet.
func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	var players []playerSummary
	err := s.db.Select(&players, `
		SELECT
			p.id,
			p.nuid,
			p.first_name,
			p.last_name,
			ttr.value AS latest_ttr,
			ttr.captured_at AS latest_ttr_at,
			qttr.value AS latest_qttr,
			qttr.captured_at AS latest_qttr_at
		FROM players p
		LEFT JOIN rating_snapshots ttr ON ttr.id = (
			SELECT id FROM rating_snapshots
			WHERE player_id = p.id AND rating_type = ?
			ORDER BY captured_at DESC LIMIT 1
		)
		LEFT JOIN rating_snapshots qttr ON qttr.id = (
			SELECT id FROM rating_snapshots
			WHERE player_id = p.id AND rating_type = ?
			ORDER BY captured_at DESC LIMIT 1
		)
		ORDER BY p.id
	`, ttrRatingType, qttrRatingType)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if players == nil {
		players = []playerSummary{}
	}

	writeJSON(w, http.StatusOK, players)
}

type historyEntry struct {
	Value      int    `json:"value" db:"value"`
	CapturedAt string `json:"captured_at" db:"captured_at"`
}

// handlePlayerHistory returns one Player's Rating snapshots of the
// requested type, oldest first.
func (s *Server) handlePlayerHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid player id")
		return
	}

	ratingType := r.URL.Query().Get("rating_type")
	if ratingType != ttrRatingType && ratingType != qttrRatingType {
		writeJSONError(w, http.StatusBadRequest, "rating_type must be TTR or QTTR")
		return
	}

	var exists bool
	if err := s.db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM players WHERE id = ?)`, id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if !exists {
		writeJSONError(w, http.StatusNotFound, "player not found")
		return
	}

	var history []historyEntry
	err = s.db.Select(&history, `
		SELECT value, captured_at FROM rating_snapshots
		WHERE player_id = ? AND rating_type = ?
		ORDER BY captured_at ASC
	`, id, ratingType)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "query failed")
		return
	}
	if history == nil {
		history = []historyEntry{}
	}

	writeJSON(w, http.StatusOK, history)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
