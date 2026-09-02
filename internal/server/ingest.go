package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

// ttrRatingType and qttrRatingType are the two Rating snapshot rows an
// Ingestion entry splits into.
const (
	ttrRatingType  = "TTR"
	qttrRatingType = "QTTR"
)

// Per-item status values in an Ingestion response.
const (
	statusOK        = "ok"
	statusDuplicate = "duplicate"
	statusError     = "error"
)

type ingestEntry struct {
	NUID       string  `json:"nuid"`
	TTR        int     `json:"ttr"`
	QTTR       int     `json:"qttr"`
	PersonName *string `json:"person_name,omitempty"`
}

type ingestResult struct {
	NUID       string `json:"nuid"`
	TTRStatus  string `json:"ttr_status"`
	QTTRStatus string `json:"qttr_status"`
	Error      string `json:"error,omitempty"`
}

type ingestResponse struct {
	Results []ingestResult `json:"results"`
}

// handleIngest turns a rating batch into stored Rating snapshot history.
// Every nuid must already correspond to a known Player; Ingestion never
// auto-creates Players.
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	var entries []ingestEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	results := make([]ingestResult, 0, len(entries))
	for _, e := range entries {
		results = append(results, s.ingestEntry(e))
	}

	writeJSON(w, http.StatusOK, ingestResponse{Results: results})
}

func (s *Server) ingestEntry(e ingestEntry) ingestResult {
	var playerID int64
	err := s.db.Get(&playerID, `SELECT id FROM players WHERE nuid = ?`, e.NUID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return ingestResult{NUID: e.NUID, TTRStatus: statusError, QTTRStatus: statusError, Error: "lookup failed"}
		}
		return ingestResult{NUID: e.NUID, TTRStatus: statusError, QTTRStatus: statusError, Error: "unknown nuid"}
	}

	return ingestResult{
		NUID:       e.NUID,
		TTRStatus:  s.insertSnapshot(playerID, ttrRatingType, e.TTR),
		QTTRStatus: s.insertSnapshot(playerID, qttrRatingType, e.QTTR),
	}
}

// insertSnapshot stores one Rating snapshot row, relying on the
// (player_id, rating_type, date(captured_at)) unique index to make a
// same-day retry a no-op rather than a duplicate row.
func (s *Server) insertSnapshot(playerID int64, ratingType string, value int) string {
	res, err := s.db.Exec(`
		INSERT OR IGNORE INTO rating_snapshots (player_id, rating_type, value)
		VALUES (?, ?, ?)
	`, playerID, ratingType, value)
	if err != nil {
		return statusError
	}

	n, err := res.RowsAffected()
	if err != nil {
		return statusError
	}
	if n == 0 {
		return statusDuplicate
	}
	return statusOK
}
