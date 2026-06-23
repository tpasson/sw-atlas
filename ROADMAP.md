# ATLAS Roadmap

This document tracks the evolution of ATLAS from a browser-only planner into a
collaborative client–server application. It is the source of truth for planned
work; each phase below maps to a **GitHub Milestone**, and the issue blocks map
to **GitHub Issues**.

> Status legend: ✅ done · 🚧 in progress · ⬜ planned

## Vision

A single, shared timeline that shows *what* ships *when* — milestones and
time-spanning events across swimlanes, with dependencies, baselines, export and
project-plan sync — in one read-only-friendly place that is a true single source
of truth for the whole team. ATLAS stays a **general-purpose** milestone planner,
not tied to any one domain.

## Phase overview

| Phase  | Goal | Status |
| ------ | ---- | ------ |
| **P0** | Foundation: Go backend + PostgreSQL + Docker, editor auth & public-read switch, frontend moved off `localStorage` | ✅ |
| **P1** | Visualization: milestone vs. event types, time-spanning bars, marker shapes | ✅ |
| **P2** | Baselines: named snapshots with switch & diff (moved / added / removed) | ✅ |
| **P3** | Export & reporting: PPTX / image export of the timeline | ⬜ |
| **P4** | Sync & drift: generic project-plan import (CSV/Excel/ICS, optional API) + schedule-drift warnings | ⬜ |

## Conventions

**Milestones:** `P0 · Foundation` … `P4 · Sync & drift`.

**Labels:**

- Area: `area:backend`, `area:frontend`, `area:db`, `area:auth`, `area:infra`, `area:docs`
- Type: `type:feature`, `type:chore`, `type:infra`, `type:bug`
- Phase: `phase:P0` … `phase:P4`
- `good first issue` where applicable

**Definition of Done (every issue):** code + tests pass, documented where
user-facing, English in code/docs/UI, no regression to the existing planner.

**Proposed repository layout:**

```
/            Vue 3 + Vite frontend (as today)
/server      Go backend (chi, pgx, goose)
/deploy      docker-compose + reverse proxy config
```

---

## P0 · Foundation

Goal: turn ATLAS into a real client–server app backed by PostgreSQL, so the plan
becomes a true single source of truth. After P0 the app behaves as today, but
data lives on the server, editing requires an editor login, and visibility can
be switched between public-read and editors-only.

### P0-1 · Backend scaffolding (Go + chi)

Set up the `/server` Go module with a chi router, structured logging,
12-factor config (env vars), and a health endpoint.

**Acceptance criteria**
- `GET /api/health` returns `200` with build/version info.
- Config (DB URL, listen address, session secret, editor credentials) is read from env.
- `go run ./server` starts locally; layout documented in `/server/README.md`.

`area:backend` · `type:infra` · `phase:P0`

### P0-2 · Database schema & migrations (goose)

Create the initial PostgreSQL schema and migration tooling.

**Tables:** `swimlane`, `sub_lane`, `item` (incl. `kind`, `marker`,
`start_date`, `end_date`, and provenance fields `source_system`,
`external_id`, `external_url`, `last_synced_at`), `link`, `app_setting`.

**Acceptance criteria**
- `goose up` / `goose down` work against a local Postgres.
- Provenance + lock fields exist on `item`.
- `app_setting` seeded with `public_read_enabled = true`.

`area:db` · `type:infra` · `phase:P0`

### P0-3 · Data access layer (sqlc + pgx)

Generate typed query functions for CRUD of swimlanes, sub-lanes, items and links.

**Acceptance criteria**
- `sqlc generate` produces typed Go code from SQL.
- Repository functions for create/read/update/delete of each entity.
- Basic unit tests against a test database (or testcontainers).

`area:backend` · `area:db` · `type:feature` · `phase:P0`

### P0-4 · Read API

Expose read endpoints returning the full plan in the shape the frontend expects.

**Acceptance criteria**
- `GET /api/plan?year=YYYY` returns swimlanes, sub-lanes, items, links as JSON.
- Response shape maps 1:1 to the current `useAppStore.js` structure.
- Year filtering works.

`area:backend` · `type:feature` · `phase:P0`

### P0-5 · Write API (with source-is-master lock)

Add create/update/delete endpoints for swimlanes, sub-lanes, items and links.

**Acceptance criteria**
- Full CRUD for each entity.
- Items with `source_system != null` reject any write with `409 Conflict`
  (the source system is always master; synced items are read-only in ATLAS).
- Deleting a swimlane/sub-lane cascades to its items and links (as today).

`area:backend` · `type:feature` · `phase:P0`

### P0-6 · Editor authentication

Single editor login backed by a bcrypt password hash; protect all write endpoints.

**Acceptance criteria**
- `POST /api/login` verifies credentials and issues a session/JWT; `POST /api/logout` clears it.
- Auth middleware guards all write endpoints (401 when missing/invalid).
- Password hash configured via env (consistent with the `hashpw` pattern), never stored in plaintext.
- Auth is behind an interface so SSO/OIDC can be added later without touching features.

`area:backend` · `area:auth` · `type:feature` · `phase:P0`

### P0-7 · Public-read switch

Global `public_read_enabled` flag controlling anonymous read access.

**Acceptance criteria**
- When `true` (default): anonymous users can call read endpoints.
- When `false`: read endpoints require an authenticated editor (401 otherwise).
- Editors can toggle the flag via `PUT /api/settings/public-read`.
- Enforced server-side, not just in the UI.

`area:backend` · `area:auth` · `type:feature` · `phase:P0`

### P0-8 · Seed / data import

Port the existing in-app seed (`src/stores/useAppStore.js`) into the database.

**Acceptance criteria**
- A `seed` command/script populates Postgres with the current demo swimlanes,
  sub-lanes, milestones and links.
- Idempotent or clearly guarded against double-seeding.

`area:backend` · `area:db` · `type:chore` · `phase:P0`

### P0-9 · Frontend: API client & store refactor

Replace `localStorage` persistence with backend calls; keep the reactive store as a client cache.

**Acceptance criteria**
- A small API client (fetch wrapper) with error handling.
- `useAppStore.js` loads from `GET /api/plan` and persists changes via the write API.
- `localStorage` is no longer the source of truth (may remain as an offline/cache hint only).
- Existing features (zoom, year nav, links, tooltips, modals) still work.

`area:frontend` · `type:feature` · `phase:P0`

### P0-10 · Frontend: auth UI & read-only enforcement

Login modal, conditional edit affordances, and the public-read toggle control.

**Acceptance criteria**
- Logged-out users see a read-only UI (no add/edit/delete controls).
- Editor login unlocks editing; logout returns to read-only.
- Public-read toggle exposed to editors in the manage UI.
- Synced (locked) items show a lock indicator and a "synced from <source>" badge.

`area:frontend` · `area:auth` · `type:feature` · `phase:P0`

### P0-11 · Docker Compose deployment

Containerize the stack for self-hosting (LAN) with a path to corporate infra.

**Acceptance criteria**
- Multi-stage `Dockerfile` builds a static Go binary; frontend built and served.
- `deploy/docker-compose.yml` brings up `app` + `postgres` (+ optional `caddy` reverse proxy/TLS).
- `.env.example` documents all config; secrets stay out of the repo.
- `docker compose up` yields a working app on a fresh machine.

`area:infra` · `type:infra` · `phase:P0`

### P0-12 · CI pipeline (nice to have)

GitHub Actions for build/test/lint on pull requests.

**Acceptance criteria**
- Backend: `go build` + `go test` + `go vet`/lint.
- Frontend: install + `npm run build`.
- Runs on PRs to `main`.

`area:infra` · `type:infra` · `phase:P0` · `good first issue`

---

## P2 · Baselines ✅ (shipped in v1.0.0)

- `baseline` snapshots of the plan (item dates + structure) with a name and timestamp.
- Switch between any baseline and the live plan; viewing a baseline is read-only.
- Diff against the live plan: **added / moved / removed** items, with counts and hover lists.

## P3–P4 (outline)

These will be expanded into detailed issues when their phase starts.

**P3 · Export & reporting**
- Client-side PPTX export (pptxgenjs) of the timeline (per swimlane / year).
- Image (PNG) export of the current view for quick sharing.

**P4 · Sync & drift**
- Generic, source-agnostic import: CSV / Excel / ICS (plus an optional REST adapter),
  mapping external items ↔ ATLAS items; imported items stay source-is-master (locked).
- Scheduled refresh + drift detection → in-app "check schedule" warnings when dates shift.

> Note: earlier domain-specific phases were dropped to keep ATLAS general-purpose.
