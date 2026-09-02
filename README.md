# TTR

Tracks table-tennis rating points for one club's roster, scraped from mytischtennis.de behind a login and viewed over time.

See [CONTEXT.md](CONTEXT.md) for the domain model and terminology.

## Server

A Go binary (`cmd/server`) that runs database migrations, then serves the HTTP API:

- [chi](https://github.com/go-chi/chi) for routing
- [sqlx](https://github.com/jmoiron/sqlx) over SQLite (`modernc.org/sqlite`, pure Go, no CGO)
- [goose](https://github.com/pressly/goose) for migrations, embedded in the binary and run automatically on startup, before the HTTP server binds its port. A failed migration exits the process non-zero.

### Running locally

```bash
cp .env.example .env
go run ./cmd/server
```

Config is read from the environment (`.env` is loaded automatically if present; see `.env.example` for the full list):

| Variable        | Default        | Purpose                              |
| ---------------- | -------------- | ------------------------------------- |
| `TTR_DB_PATH`     | `/data/ttr.db` | Path to the SQLite database file      |
| `TTR_ADDR`        | `:8080`        | Address the HTTP server listens on    |
| `INGESTION_KEY`   | —              | Static key for Ingestion requests (not yet enforced) |

Check it's up:

```bash
curl http://localhost:8080/health
```

### Tests

```bash
go build ./...
go vet ./...
go test ./...
```

## Deployment

Deployed to Fly.io as `ttr-server` (see [fly.toml](fly.toml)). Every push to `main` runs the GitHub Actions workflow in [.github/workflows/deploy.yml](.github/workflows/deploy.yml): build, vet, test, then `flyctl deploy`.

To deploy manually:

```bash
fly deploy
```
