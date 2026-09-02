package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// UpsertPlayer inserts a player row or, if keyColumn/keyValue already match
// an existing row, refreshes its name in place. keyColumn must be a column
// the players table carries a unique index on ("nuid" or "internal_id") --
// it's trusted, code-supplied SQL, never user input.
func UpsertPlayer(db *sqlx.DB, keyColumn, keyValue, firstName, lastName string) error {
	query := fmt.Sprintf(`
		INSERT INTO players (%[1]s, first_name, last_name)
		VALUES (?, ?, ?)
		ON CONFLICT(%[1]s) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name
	`, keyColumn)
	_, err := db.Exec(query, keyValue, firstName, lastName)
	return err
}
