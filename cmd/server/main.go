// Command server boots the TTR API: it runs database migrations, then
// serves HTTP traffic.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/roster"
	"github.com/iliksis/ttr/internal/server"
	"github.com/joho/godotenv"
)

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

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, server.New(db, ingestionKey)); err != nil {
		log.Fatalf("server: %v", err)
	}
}
