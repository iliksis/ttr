// Package roster holds the manually-maintained list of club Players and
// seeds it into the database at startup, since club-derived roster sourcing
// doesn't exist yet.
package roster

import (
	"fmt"

	"github.com/iliksis/ttr/internal/database"
	"github.com/jmoiron/sqlx"
)

// ManualPlayer is one manually-added Player, known upfront by the admin.
type ManualPlayer struct {
	NUID      string
	FirstName string
	LastName  string
}

// Manual is the static, server-side config list of manually-added Players.
// Redeploy with an updated list to change the roster; there's no admin UI.
var Manual = []ManualPlayer{}

// Seed upserts every entry in players into the players table, keyed by nuid.
// Safe to call on every startup: an existing Player's name is refreshed
// in place rather than duplicated.
func Seed(db *sqlx.DB, players []ManualPlayer) error {
	for _, p := range players {
		if err := database.UpsertPlayer(db, "nuid", p.NUID, p.FirstName, p.LastName); err != nil {
			return fmt.Errorf("seed player %s: %w", p.NUID, err)
		}
	}
	return nil
}
