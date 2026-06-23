# ATLAS backend

Go service that exposes the ATLAS REST API, talks to PostgreSQL, and (optionally)
serves the built frontend so the whole app ships as a single container.

## Stack

- **chi** — HTTP router
- **pgx** (`pgxpool`) — PostgreSQL access
- **goose** — embedded schema migrations (run automatically on startup)
- **golang-jwt** + **bcrypt** — editor authentication

## Layout

```
server/
├─ cmd/atlas/main.go        # entry point + subcommands
└─ internal/
   ├─ config/               # env-based configuration
   ├─ db/                   # pgx pool + embedded migrations
   │  └─ migrations/        # goose .sql files
   ├─ store/                # typed data-access layer (CRUD, source-is-master lock)
   ├─ auth/                 # bcrypt verify + JWT cookie
   ├─ api/                  # routes, middleware, handlers
   └─ seed/                 # demo data importer
```

## Commands

```bash
atlas serve         # start the HTTP server (default)
atlas migrate       # apply pending migrations
atlas migrate-down  # roll back the latest migration
atlas seed          # load demo data (idempotent)
atlas hashpw <pw>   # print a bcrypt hash for ATLAS_EDITOR_PASSWORD_HASH
```

## Configuration (environment variables)

| Variable | Default | Purpose |
| --- | --- | --- |
| `ATLAS_LISTEN_ADDR` | `:8080` | Listen address |
| `ATLAS_DATABASE_URL` | `postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable` | PostgreSQL DSN |
| `ATLAS_SESSION_SECRET` | `dev-insecure-change-me` | JWT signing secret |
| `ATLAS_EDITOR_USERNAME` | `editor` | Editor login name |
| `ATLAS_EDITOR_PASSWORD_HASH` | _(empty)_ | bcrypt hash; empty disables login |
| `ATLAS_EDITOR_PASSWORD_HASH_FILE` | _(empty)_ | path to a file with the bcrypt hash (used by the Docker secret); fallback when the env var above is empty |
| `ATLAS_STATIC_DIR` | _(empty)_ | If set, serve the built SPA from this dir |

## Local development

```bash
# 1. A PostgreSQL to point at (any will do):
docker run -d --name atlas-pg -e POSTGRES_USER=atlas -e POSTGRES_PASSWORD=atlas \
  -e POSTGRES_DB=atlas -p 5432:5432 postgres:16-alpine

# 2. Set an editor password and run:
export ATLAS_EDITOR_PASSWORD_HASH="$(go run ./cmd/atlas hashpw 'secret')"
go run ./cmd/atlas seed     # optional demo data
go run ./cmd/atlas serve
```

Then run the frontend dev server from the repo root with the API proxied:

```bash
VITE_API_TARGET=http://localhost:8080 npm run dev
```

## API surface

- `GET  /api/health`
- `POST /api/login` · `POST /api/logout` · `GET /api/me`
- `GET  /api/plan` *(public when public-read is on, else editor-only)*
- `GET/PUT /api/settings/public-read`
- `POST/PUT/DELETE /api/swimlanes[/{id}]`, `POST /api/swimlanes/{id}/move`, `POST /api/swimlanes/{id}/sublanes`
- `PUT/DELETE /api/sublanes/{id}`
- `POST/PUT/DELETE /api/items[/{id}]` — writes to source-managed items return **409**
- `POST/DELETE /api/links`

All write endpoints require an editor session.
