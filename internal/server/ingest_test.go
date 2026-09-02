package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iliksis/ttr/internal/server"
	"github.com/jmoiron/sqlx"
)

const testIngestionKey = "test-key"

func seedPlayer(t *testing.T, db *sqlx.DB, nuid string) int64 {
	t.Helper()

	res, err := db.Exec(`INSERT INTO players (nuid, first_name, last_name) VALUES (?, ?, ?)`, nuid, "First", "Last")
	if err != nil {
		t.Fatalf("seed player: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed player id: %v", err)
	}
	return id
}

func ingestRequest(t *testing.T, handler http.Handler, key string, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/ingest", strings.NewReader(body))
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestIngest_RejectsMissingKey(t *testing.T) {
	db := testDB(t)
	handler := server.New(db, testIngestionKey)

	rec := ingestRequest(t, handler, "", `[]`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestIngest_RejectsInvalidKey(t *testing.T) {
	db := testDB(t)
	handler := server.New(db, testIngestionKey)

	rec := ingestRequest(t, handler, "wrong-key", `[]`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestIngest_ValidBatchCreatesSnapshots(t *testing.T) {
	db := testDB(t)
	playerID := seedPlayer(t, db, "known-1")
	handler := server.New(db, testIngestionKey)

	body := `[{"nuid":"known-1","ttr":1500,"qttr":1450}]`

	rec := ingestRequest(t, handler, testIngestionKey, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM rating_snapshots WHERE player_id = ?`, playerID); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if count != 2 {
		t.Fatalf("snapshot count = %d, want 2", count)
	}

	var capturedAt string
	if err := db.Get(&capturedAt, `SELECT captured_at FROM rating_snapshots WHERE player_id = ? AND rating_type = 'TTR'`, playerID); err != nil {
		t.Fatalf("captured_at query error = %v, want nil", err)
	}
	if capturedAt == "" {
		t.Fatal("captured_at is empty, want a server-assigned timestamp")
	}
}

func TestIngest_UnknownNUIDRejectedPerItem(t *testing.T) {
	db := testDB(t)
	knownID := seedPlayer(t, db, "known-1")
	handler := server.New(db, testIngestionKey)

	body := `[
		{"nuid":"unknown-1","ttr":1500,"qttr":1450},
		{"nuid":"known-1","ttr":1600,"qttr":1550}
	]`

	rec := ingestRequest(t, handler, testIngestionKey, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Results []struct {
			NUID       string `json:"nuid"`
			TTRStatus  string `json:"ttr_status"`
			QTTRStatus string `json:"qttr_status"`
			Error      string `json:"error"`
		} `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(resp.Results))
	}

	unknown, known := resp.Results[0], resp.Results[1]
	if unknown.Error == "" {
		t.Fatalf("unknown-1 result = %+v, want a non-empty error", unknown)
	}
	if known.Error != "" {
		t.Fatalf("known-1 result = %+v, want no error", known)
	}

	var playerCount int
	if err := db.Get(&playerCount, `SELECT COUNT(*) FROM players WHERE nuid = ?`, "unknown-1"); err != nil {
		t.Fatalf("player count query error = %v, want nil", err)
	}
	if playerCount != 0 {
		t.Fatalf("player count for unknown-1 = %d, want 0 (Ingestion must never auto-create Players)", playerCount)
	}

	var unknownSnapshotCount int
	if err := db.Get(&unknownSnapshotCount, `SELECT COUNT(*) FROM rating_snapshots`); err != nil {
		t.Fatalf("snapshot count query error = %v, want nil", err)
	}

	var knownSnapshotCount int
	if err := db.Get(&knownSnapshotCount, `SELECT COUNT(*) FROM rating_snapshots WHERE player_id = ?`, knownID); err != nil {
		t.Fatalf("known snapshot count query error = %v, want nil", err)
	}
	if knownSnapshotCount != 2 {
		t.Fatalf("known-1 snapshot count = %d, want 2", knownSnapshotCount)
	}
	if unknownSnapshotCount != knownSnapshotCount {
		t.Fatalf("total snapshot count = %d, want %d (unknown entry must create no rows)", unknownSnapshotCount, knownSnapshotCount)
	}
}

func TestIngest_SameDayRetryIsIdempotent(t *testing.T) {
	db := testDB(t)
	playerID := seedPlayer(t, db, "known-1")
	handler := server.New(db, testIngestionKey)

	body := `[{"nuid":"known-1","ttr":1500,"qttr":1450}]`

	if rec := ingestRequest(t, handler, testIngestionKey, body); rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec := ingestRequest(t, handler, testIngestionKey, body); rec.Code != http.StatusOK {
		t.Fatalf("retry status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM rating_snapshots WHERE player_id = ?`, playerID); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if count != 2 {
		t.Fatalf("snapshot count after retry = %d, want 2 (no duplicates)", count)
	}
}
