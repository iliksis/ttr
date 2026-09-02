// Package clubroster automatically and repeatedly sources the club's roster
// from mytischtennis.de, so Players don't have to be hand-entered to show
// up in the system.
package clubroster

import (
	"context"
	"fmt"

	"github.com/iliksis/ttr/internal/mytt"
	"github.com/jmoiron/sqlx"
)

// teamsPlayersFetcher is the subset of *mytt.Client the Sync job needs,
// so tests can stub it without a real HTTP server.
type teamsPlayersFetcher interface {
	FetchTeams(ctx context.Context, clubNumber, organization string) ([]mytt.Team, error)
	FetchTeamPlayers(ctx context.Context, teamID string) ([]mytt.Player, error)
}

// Sync fetches every team under clubNumber/organization and every player on
// each of those teams, unconditionally, then upserts each player into the
// players table by internal_id. It never resolves nuid and never deletes a
// Player: one absent from this run's fetch simply isn't touched, keeping
// its row and Rating snapshot history intact.
func Sync(ctx context.Context, db *sqlx.DB, client teamsPlayersFetcher, clubNumber, organization string) error {
	teams, err := client.FetchTeams(ctx, clubNumber, organization)
	if err != nil {
		return fmt.Errorf("fetch teams: %w", err)
	}

	for _, team := range teams {
		players, err := client.FetchTeamPlayers(ctx, team.TeamID)
		if err != nil {
			return fmt.Errorf("fetch team %s players: %w", team.TeamID, err)
		}

		for _, p := range players {
			// An empty internal_id would collide with every other such
			// entry under the unique index, overwriting an unrelated
			// player's name; skip rather than risk that.
			if p.InternalID == "" {
				continue
			}
			if err := upsertPlayer(db, p); err != nil {
				return fmt.Errorf("upsert player %s: %w", p.InternalID, err)
			}
		}
	}

	return nil
}

func upsertPlayer(db *sqlx.DB, p mytt.Player) error {
	_, err := db.Exec(`
		INSERT INTO players (internal_id, first_name, last_name)
		VALUES (?, ?, ?)
		ON CONFLICT(internal_id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name
	`, p.InternalID, p.FirstName, p.LastName)
	return err
}
