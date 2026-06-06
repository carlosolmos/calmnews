# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build local binary
make build                  # outputs to bin/calmnews
go build -o bin/calmnews ./cmd/calmnews

# Build Linux binary (for deployment)
make build-linux            # GOOS=linux GOARCH=amd64

# Run locally
./bin/calmnews

# Docker
make docker-build           # builds calmnews:latest image
make docker-up              # docker compose up -d
make docker-down

# No test suite exists in this project
go vet ./...                # static analysis
```

## Architecture

CalmNews is a single-binary Go RSS reader. The binary embeds all HTML templates and CSS at compile time (via `//go:embed` in `internal/web/templates.go`), so there are no runtime file dependencies beyond the data directory.

**Startup flow** (`cmd/calmnews/main.go`):
1. Resolves data dir (`~/.calmnews/` or `$CALMNEWS_DATA_DIR`)
2. Loads or creates `config.yaml` in data dir
3. Initializes SQLite at `news.db` in data dir and runs migrations
4. Syncs feeds from config → DB via upsert
5. Starts background scheduler goroutine (fetches immediately, then on interval)
6. Starts HTTP server (default `0.0.0.0:8080`, overridable via `$CALMNEWS_LISTEN_ADDR`)

**Package responsibilities:**
- `internal/config` — YAML config load/save; `DataDir()` checks `$CALMNEWS_DATA_DIR` then `~/.calmnews/`
- `internal/storage` — SQLite schema (migrations in `RunMigrations`), all DB access functions. Article primary key is `SHA256(feedURL + "|" + entryGUID)`. Secondary duplicate check by title at fetch time.
- `internal/feeds` — `FetchFeed` (HTTP GET) + `ParseFeed` (gofeed) + `StartScheduler` (goroutine with ticker). The global refresh interval is read from `cfg.Feeds[0].RefreshIntervalMinutes`; per-feed intervals are honored inside `fetchAllFeeds`.
- `internal/filter` — Blocklist filtering: case-insensitive substring match against `title + " " + summary`
- `internal/web` — `Server` struct holds `*sql.DB`, `*config.Config`, and `configPath`. Settings writes go directly to both the in-memory config and `config.yaml`. Templates are parsed on every request (no caching).

**Config/DB relationship:** Feeds exist in both `config.yaml` and the `feeds` table. On startup, config is the source of truth and syncs to DB. Settings changes (add feed, toggle enabled, update blocklist) update both in-memory config and write `config.yaml`, then update the DB.

**Article lifecycle:** Fetched articles are upserted (on-conflict preserves `is_read`/`is_saved`). The scheduler deletes non-saved articles older than 72 hours after each fetch cycle. Saved articles (`is_saved = 1`) are never expired.

**Blocklist filtering** happens at query time in the HTTP handler, not at storage time — all articles are stored regardless of the blocklist.

## Deployment

Production runs behind Traefik (see `traefik/docker-compose.yml`). The `/settings` path is protected by Traefik basic auth middleware — there is no application-level auth. Replace `<CREDENTIALS>` in the compose file with a bcrypt-hashed htpasswd string before deploying.

Data is persisted at `/opt/calmnews/data` on the host, mounted into the container at `/app/data`.
