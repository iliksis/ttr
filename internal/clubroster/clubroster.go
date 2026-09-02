// Package clubroster automatically and repeatedly sources the club's roster
// from mytischtennis.de, so Players don't have to be hand-entered to show
// up in the system.
package clubroster

import (
	"context"
	"fmt"
	"log"

	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/mytt"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

// teamsPlayersFetcher is the subset of *mytt.Client the Sync job needs,
// so tests can stub it without a real HTTP server.
type teamsPlayersFetcher interface {
	FetchTeams(ctx context.Context, clubNumber, organization string) ([]mytt.Team, error)
	FetchTeamPlayers(ctx context.Context, teamID string) ([]mytt.Player, error)
}

// maxConcurrentTeamFetches bounds how many FetchTeamPlayers calls run at
// once, so a club with many teams doesn't open dozens of simultaneous
// connections to mytischtennis.de.
const maxConcurrentTeamFetches = 5

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

	teamPlayers := make([][]mytt.Player, len(teams))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentTeamFetches)
	for i, team := range teams {
		g.Go(func() error {
			players, err := client.FetchTeamPlayers(gctx, team.TeamID)
			if err != nil {
				return fmt.Errorf("fetch team %s players: %w", team.TeamID, err)
			}
			teamPlayers[i] = players
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	for i, team := range teams {
		for _, p := range teamPlayers[i] {
			// An empty internal_id would collide with every other such
			// entry under the unique index, overwriting an unrelated
			// player's name; skip rather than risk that.
			if p.InternalID == "" {
				log.Printf("club roster sync: skipping player with empty internal_id (team %s, %s %s)", team.TeamID, p.FirstName, p.LastName)
				continue
			}
			if err := database.UpsertPlayer(db, "internal_id", p.InternalID, p.FirstName, p.LastName); err != nil {
				return fmt.Errorf("upsert player %s: %w", p.InternalID, err)
			}
		}
	}

	return nil
}
