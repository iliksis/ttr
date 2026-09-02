package database_test

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/iliksis/ttr/internal/database"
)

func TestOpen_RunsMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ttr.db")

	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping() error = %v, want nil", err)
	}
}

func TestOpen_BrokenMigrationFailsClosed(t *testing.T) {
	broken := fstest.MapFS{
		"00001_broken.sql": &fstest.MapFile{Data: []byte(`
-- +goose Up
-- +goose StatementBegin
THIS IS NOT VALID SQL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
`)},
	}

	dbPath := filepath.Join(t.TempDir(), "ttr.db")

	db, err := database.Open(dbPath, broken)
	if err == nil {
		db.Close()
		t.Fatal("Open() error = nil, want non-nil for a broken migration")
	}
}
