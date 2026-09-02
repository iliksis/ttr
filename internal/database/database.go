// Package database opens the TTR SQLite database and runs its goose migrations.
package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// Migrations is the migration set baked into the binary.
var Migrations fs.FS = mustSub(embeddedMigrations, "migrations")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// busyTimeoutDSNSuffix makes concurrent writers (e.g. an Ingestion request
// landing while the club roster fetch job is mid-upsert) block and retry
// instead of failing immediately with SQLITE_BUSY.
const busyTimeoutDSNSuffix = "?_pragma=busy_timeout(5000)"

// Open opens the SQLite database at path and runs migrations against it
// before returning. A migration failure closes the connection and returns
// a non-nil error.
func Open(path string, migrations fs.FS) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", path+busyTimeoutDSNSuffix)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := Migrate(db.DB, migrations); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// Migrate runs every pending migration in migrations against db.
func Migrate(db *sql.DB, migrations fs.FS) error {
	goose.SetBaseFS(migrations)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
