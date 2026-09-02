// Command server boots the TTR API: it runs database migrations, then
// serves HTTP traffic.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/iliksis/ttr/internal/clubroster"
	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/mytt"
	"github.com/iliksis/ttr/internal/roster"
	"github.com/iliksis/ttr/internal/server"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

// clubRosterSyncInterval is how often the club roster sync job re-fetches
// the club's teams and players; the job also runs once at startup.
const clubRosterSyncInterval = 24 * time.Hour

func main() {
	// Loads .env into the environment for local dev; a missing file is not
	// an error, since production config comes from Fly secrets instead.
	_ = godotenv.Load()

	dbPath := os.Getenv("TTR_DB_PATH")
	if dbPath == "" {
		dbPath = "/data/ttr.db"
	}

	addr := os.Getenv("TTR_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ingestionKey := os.Getenv("INGESTION_KEY")
	if ingestionKey == "" {
		log.Fatal("INGESTION_KEY must be set")
	}

	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	defer db.Close()

	if err := roster.Seed(db, roster.Manual); err != nil {
		log.Fatalf("seed roster: %v", err)
	}

	startClubRosterSync(db)

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, server.New(db, ingestionKey)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// startClubRosterSync launches the daily club roster sync job in the
// background when CLUB_NUMBER is configured. It's opt-in: without a club
// number there's no club to sync against, so the job is skipped entirely.
func startClubRosterSync(db *sqlx.DB) {
	clubNumber := os.Getenv("CLUB_NUMBER")
	organization := os.Getenv("CLUB_ORGANIZATION")

	if clubNumber == "" && organization == "" {
		log.Print("CLUB_NUMBER not set, skipping club roster sync")
		return
	}
	if clubNumber == "" {
		log.Print("WARNING: CLUB_ORGANIZATION is set but CLUB_NUMBER is not; skipping club roster sync. Set both to enable it.")
		return
	}
	if organization == "" {
		log.Print("WARNING: CLUB_NUMBER is set but CLUB_ORGANIZATION is not; skipping club roster sync. Set both to enable it.")
		return
	}

	client := mytt.NewClient("")

	go clubroster.RunDaily(context.Background(), clubRosterSyncInterval, func(ctx context.Context) {
		log.Printf("club roster sync: starting (club %s/%s)", organization, clubNumber)
		if err := clubroster.Sync(ctx, db, client, clubNumber, organization); err != nil {
			log.Printf("club roster sync: %v", err)
			return
		}
		log.Print("club roster sync: done")
	})
}
