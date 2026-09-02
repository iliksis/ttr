package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/iliksis/ttr/internal/server"
	"github.com/jmoiron/sqlx"
)

func seedManualPlayer(t *testing.T, db *sqlx.DB, firstName, lastName string) int64 {
	t.Helper()

	res, err := db.Exec(`INSERT INTO players (internal_id, first_name, last_name) VALUES (?, ?, ?)`, "manual-"+firstName, firstName, lastName)
	if err != nil {
		t.Fatalf("seed manual player: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed manual player id: %v", err)
	}
	return id
}

func seedSnapshot(t *testing.T, db *sqlx.DB, playerID int64, ratingType string, value int, capturedAt string) {
	t.Helper()

	_, err := db.Exec(`INSERT INTO rating_snapshots (player_id, rating_type, value, captured_at) VALUES (?, ?, ?, ?)`,
		playerID, ratingType, value, capturedAt)
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
}

func getJSON(t *testing.T, handler http.Handler, path string, out any) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
			t.Fatalf("decode response: %v, body = %s", err, rec.Body.String())
		}
	}
	return rec
}

func TestRoster_OmitsPlayersWithoutNUID(t *testing.T) {
	db := testDB(t)
	seedPlayer(t, db, "with-nuid")
	seedManualPlayer(t, db, "No", "Nuid")
	handler := server.New(db, testIngestionKey)

	var roster []struct {
		NUID string `json:"nuid"`
	}
	rec := getJSON(t, handler, "/api/roster", &roster)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(roster) != 1 {
		t.Fatalf("roster = %+v, want 1 entry", roster)
	}
	if roster[0].NUID != "with-nuid" {
		t.Fatalf("roster[0].NUID = %q, want %q", roster[0].NUID, "with-nuid")
	}
}

func TestPlayers_IncludesLatestSnapshotsAndNullsWhenAbsent(t *testing.T) {
	db := testDB(t)
	withHistory := seedPlayer(t, db, "has-history")
	seedSnapshot(t, db, withHistory, "TTR", 1500, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, withHistory, "TTR", 1550, "2026-01-02T00:00:00.000Z")
	seedSnapshot(t, db, withHistory, "QTTR", 1400, "2026-01-01T00:00:00.000Z")
	noHistory := seedManualPlayer(t, db, "No", "History")
	handler := server.New(db, testIngestionKey)

	var players []struct {
		ID           int64   `json:"id"`
		NUID         *string `json:"nuid"`
		FirstName    string  `json:"first_name"`
		LastName     string  `json:"last_name"`
		LatestTTR    *int    `json:"latest_ttr"`
		LatestTTRAt  *string `json:"latest_ttr_at"`
		LatestQTTR   *int    `json:"latest_qttr"`
		LatestQTTRAt *string `json:"latest_qttr_at"`
	}
	rec := getJSON(t, handler, "/api/players", &players)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(players) != 2 {
		t.Fatalf("players = %+v, want 2 entries", players)
	}

	var withHistoryEntry, noHistoryEntry *struct {
		ID           int64   `json:"id"`
		NUID         *string `json:"nuid"`
		FirstName    string  `json:"first_name"`
		LastName     string  `json:"last_name"`
		LatestTTR    *int    `json:"latest_ttr"`
		LatestTTRAt  *string `json:"latest_ttr_at"`
		LatestQTTR   *int    `json:"latest_qttr"`
		LatestQTTRAt *string `json:"latest_qttr_at"`
	}
	for i := range players {
		switch players[i].ID {
		case withHistory:
			withHistoryEntry = &players[i]
		case noHistory:
			noHistoryEntry = &players[i]
		}
	}
	if withHistoryEntry == nil || noHistoryEntry == nil {
		t.Fatalf("players = %+v, want entries for both seeded ids", players)
	}

	if withHistoryEntry.LatestTTR == nil || *withHistoryEntry.LatestTTR != 1550 {
		t.Fatalf("with-history latest_ttr = %v, want 1550 (most recent snapshot)", withHistoryEntry.LatestTTR)
	}
	if withHistoryEntry.LatestQTTR == nil || *withHistoryEntry.LatestQTTR != 1400 {
		t.Fatalf("with-history latest_qttr = %v, want 1400", withHistoryEntry.LatestQTTR)
	}

	if noHistoryEntry.NUID != nil {
		t.Fatalf("no-history nuid = %v, want nil", noHistoryEntry.NUID)
	}
	if noHistoryEntry.LatestTTR != nil || noHistoryEntry.LatestTTRAt != nil {
		t.Fatalf("no-history latest_ttr/_at = %v/%v, want nil/nil", noHistoryEntry.LatestTTR, noHistoryEntry.LatestTTRAt)
	}
	if noHistoryEntry.LatestQTTR != nil || noHistoryEntry.LatestQTTRAt != nil {
		t.Fatalf("no-history latest_qttr/_at = %v/%v, want nil/nil", noHistoryEntry.LatestQTTR, noHistoryEntry.LatestQTTRAt)
	}
}

func TestPlayerHistory_ReturnsRequestedTypeOldestFirst(t *testing.T) {
	db := testDB(t)
	playerID := seedPlayer(t, db, "known-1")
	seedSnapshot(t, db, playerID, "TTR", 1500, "2026-01-02T00:00:00.000Z")
	seedSnapshot(t, db, playerID, "TTR", 1400, "2026-01-01T00:00:00.000Z")
	seedSnapshot(t, db, playerID, "QTTR", 1300, "2026-01-01T00:00:00.000Z")
	handler := server.New(db, testIngestionKey)

	var history []struct {
		Value      int    `json:"value"`
		CapturedAt string `json:"captured_at"`
	}
	rec := getJSON(t, handler, "/api/players/"+strconv.FormatInt(playerID, 10)+"/history?rating_type=TTR", &history)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(history) != 2 {
		t.Fatalf("history = %+v, want 2 entries", history)
	}
	if history[0].Value != 1400 || history[1].Value != 1500 {
		t.Fatalf("history values = [%d, %d], want [1400, 1500] (oldest to newest)", history[0].Value, history[1].Value)
	}
}

func TestPlayerHistory_UnknownPlayerReturnsNotFound(t *testing.T) {
	db := testDB(t)
	handler := server.New(db, testIngestionKey)

	req := httptest.NewRequest(http.MethodGet, "/api/players/999/history?rating_type=TTR", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestPlayerHistory_InvalidRatingTypeReturnsBadRequest(t *testing.T) {
	db := testDB(t)
	playerID := seedPlayer(t, db, "known-1")
	handler := server.New(db, testIngestionKey)

	req := httptest.NewRequest(http.MethodGet, "/api/players/"+strconv.FormatInt(playerID, 10)+"/history?rating_type=BOGUS", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
