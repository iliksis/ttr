// Command server boots the TTR API: it runs database migrations, then
// serves HTTP traffic.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/iliksis/ttr/internal/database"
	"github.com/iliksis/ttr/internal/server"
)

func main() {
	dbPath := os.Getenv("TTR_DB_PATH")
	if dbPath == "" {
		dbPath = "/data/ttr.db"
	}

	addr := os.Getenv("TTR_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// INGESTION_KEY is wired through for later ingestion-endpoint tickets;
	// not read anywhere yet.
	_ = os.Getenv("INGESTION_KEY")

	db, err := database.Open(dbPath, database.Migrations)
	if err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	defer db.Close()

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, server.New()); err != nil {
		log.Fatalf("server: %v", err)
	}
}
