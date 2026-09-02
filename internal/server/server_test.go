package server_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/server"
	"github.com/jmoiron/sqlx"
)

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ttr.db")
	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}

// insertPlayer inserts a row into players, leaving nuid or internalID unset
// (NULL) when its Valid field is false.
func insertPlayer(t *testing.T, db *sqlx.DB, nuid, internalID sql.NullString, firstName, lastName string) int64 {
	t.Helper()

	res, err := db.Exec(`INSERT INTO players (nuid, internal_id, first_name, last_name) VALUES (?, ?, ?, ?)`,
		nuid, internalID, firstName, lastName)
	if err != nil {
		t.Fatalf("insert player: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("insert player id: %v", err)
	}
	return id
}

func TestHealth_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.New(testDB(t), "test-key").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	got := rec.Body.String()
	want := `{"status":"ok"}`
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
