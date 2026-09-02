package roster_test

import (
	"path/filepath"
	"testing"

	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/roster"
)

func TestSeed_CreatesKnownPlayers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ttr.db")
	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	defer db.Close()

	players := []roster.ManualPlayer{
		{NUID: "nuid-1", FirstName: "Ada", LastName: "Lovelace"},
	}

	if err := roster.Seed(db, players); err != nil {
		t.Fatalf("Seed() error = %v, want nil", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM players WHERE nuid = ?`, "nuid-1"); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if count != 1 {
		t.Fatalf("player count = %d, want 1", count)
	}
}

func TestSeed_IsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ttr.db")
	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}
	defer db.Close()

	players := []roster.ManualPlayer{
		{NUID: "nuid-1", FirstName: "Ada", LastName: "Lovelace"},
	}

	if err := roster.Seed(db, players); err != nil {
		t.Fatalf("Seed() [1] error = %v, want nil", err)
	}
	if err := roster.Seed(db, players); err != nil {
		t.Fatalf("Seed() [2] error = %v, want nil", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM players WHERE nuid = ?`, "nuid-1"); err != nil {
		t.Fatalf("count query error = %v, want nil", err)
	}
	if count != 1 {
		t.Fatalf("player count = %d, want 1 after re-seeding", count)
	}
}
