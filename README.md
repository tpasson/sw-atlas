<div align="center">

# 🛰️ ATLAS

**Collaborative milestone & roadmap planning — date-anchored timelines, events, baselines and groups.**

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Vue 3](https://img.shields.io/badge/Vue.js-3-42b883?logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](deploy/docker-compose.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#-contributing)
[![Status: Stable](https://img.shields.io/badge/status-stable-brightgreen.svg)](#-roadmap)
[![Release](https://img.shields.io/github/v/release/tpasson/sw-atlas)](https://github.com/tpasson/sw-atlas/releases)

[**🚀 Live Demo**](https://tpasson.github.io/sw-atlas/) · [Features](#-features) · [Roadmap](#%EF%B8%8F-roadmap) · [Getting Started](#-getting-started) · [Contributing](#-contributing)

</div>

---

## Overview

**ATLAS** is a visual planning tool that gives any team a single, shared picture of *what* happens *when* — across customers, events, products, platforms, features and releases.

Work is organized into **swimlanes** and **sub-lanes** on a month/year timeline. On that grid you place **milestones** (point-in-time markers) and **events** (bars that span a period), connect them with **dependency links**, and drill into the *What / Why / Where / Who / When* behind each entry.

ATLAS is **general-purpose** and not tied to any one domain: use the same timeline for product roadmaps, delivery plans, go-to-market calendars, release trains or any program of work that benefits from a shared, date-anchored view.

> **Project status:** ATLAS is a collaborative **client–server** application — a Vue SPA backed by a Go API, PostgreSQL, editor authentication and a self-hostable Docker deployment — stable as of **v1.0.0**. See the [Roadmap](#%EF%B8%8F-roadmap) for what's live today versus what's still coming (export & project-plan sync).

## ✨ Features

### Available today
- 📊 **Date-anchored timeline** — swimlanes & sub-lanes across fixed months and years, with a sticky header and lane packing
- 🧭 **Milestones** with rich detail: *What / Why / Where / Who / When*
- 🎯 **Events as bars** that span a period, with configurable markers (diamond, circle, cone, flag)
- 🔗 **Dependency links** between items, highlighted on hover
- 🏷️ **Groups** — color-coded groupings with a legend
- 🖱️ **Click-to-inspect** details; quick add/edit via modals
- 🔎 **Zoom** and year navigation
- 👁️ **Read-only mode** for sharing
- 🧬 **Baselines** — save named snapshots and compare any baseline against the live plan (added / moved / removed, with counts)
- 🗄️ **Backend + PostgreSQL** — a true single source of truth for the whole team
- 🔐 **Editor login** with a global *public-read* switch (open to everyone, or editors-only at the flip of a toggle)
- 🐳 **Self-hostable** via Docker Compose, or run the browser-only build with local persistence

### On the roadmap
- 📑 **Export & reporting** — PPTX / image export of the timeline
- 🔄 **Generic project-plan import** (CSV / Excel / ICS, optional API) — imported items stay source-is-master (locked)
- ⚠️ **Schedule-drift warnings** (in-app) when imported dates shift

## 📸 Screenshots

> The **[live demo](https://tpasson.github.io/sw-atlas/)** runs entirely
> in your browser (no backend; changes are kept in `localStorage` only). The full
> app — with the Go API, PostgreSQL and editor login — runs via Docker (see below).

<!-- ![ATLAS timeline](docs/screenshot.png) -->

## 🧱 Tech Stack

| Layer    | Stack                                                  |
| -------- | ------------------------------------------------------ |
| Frontend | Vue 3, Vite                                            |
| Backend  | Go (chi router, pgx, goose migrations)                 |
| Database | PostgreSQL 16                                          |
| Deploy   | Docker Compose (app · postgres) · GitHub Pages demo    |

> The browser-only demo build (`npm run build:demo`, GitHub Pages) keeps data in
> `localStorage` and needs no backend; the full app uses the Go API + PostgreSQL.

## 🏗️ Architecture

```
Browser (Vue SPA)
   │  HTTPS / JSON REST
   ▼
Reverse Proxy (Caddy/nginx) ── TLS, optional SSO/OIDC   [bring your own]
   ▼
API (Go · chi)
   ├─ Auth (editor login) + global "public read" switch
   ├─ Domain API (swimlanes · sub-lanes · items · links · groups)
   ├─ Baseline & diff service
   ├─ Import workers (cron) ── generic project-plan import (CSV/Excel/ICS)   [planned · P4]
   └─ Export service (PPTX / image)                                          [planned · P3]
   ▼
PostgreSQL  +  file storage (exports)
```

The collaborative stack — Vue SPA, Go API, PostgreSQL, editor auth and baselines —
is live as of **v1.0.0**. Import workers and the export service are still on the
roadmap; the reverse proxy is provided by you in front of the container.

## 🚀 Getting Started

### Install from a prebuilt image (recommended for hosting)

ATLAS is published to the **GitHub Container Registry**, so you can run it on any
host with Docker — no source checkout, no local build. The image bundles the Go
API and the built UI; it runs next to PostgreSQL via Compose.

> **Image:** `ghcr.io/tpasson/sw-atlas:latest`
> The package must be **public** to pull without authentication (GitHub → repo →
> *Packages* → the package → *Package settings* → *Change visibility* → Public).
> Otherwise run `docker login ghcr.io` with a personal access token first.

```bash
# 1. Create a working directory
mkdir atlas && cd atlas

# 2. Fetch the Compose file and the env template
curl -O  https://raw.githubusercontent.com/tpasson/sw-atlas/main/deploy/docker-compose.ghcr.yml
curl -o .env https://raw.githubusercontent.com/tpasson/sw-atlas/main/deploy/.env.example

# 3. Edit .env — set a strong POSTGRES_PASSWORD, the matching ATLAS_DATABASE_URL,
#    and ATLAS_SESSION_SECRET (e.g.  openssl rand -hex 32 )
nano .env

# 4. Create the editor login hash straight from the image
docker run --rm ghcr.io/tpasson/sw-atlas:latest hashpw 'your-password' > editor_hash.txt

# 5. Start it (pulls the image, runs app + PostgreSQL, migrates on startup)
docker compose -f docker-compose.ghcr.yml up -d

# 6. Optional: load the demo data
docker compose -f docker-compose.ghcr.yml run --rm app seed
```

ATLAS is then at `http://<host>:8080` (or `ATLAS_PORT`). It is publicly
**read-only** by default — log in with `ATLAS_EDITOR_USERNAME` + your password to
edit, or to switch it to editors-only.

**Update later:**
```bash
docker compose -f docker-compose.ghcr.yml pull && docker compose -f docker-compose.ghcr.yml up -d
```

**Public deployment:** put a TLS-terminating reverse proxy (Caddy, Traefik, nginx)
in front — the container serves plain HTTP on its port. A specific version can be
pinned by replacing `:latest` with a tag like `:v1.0.0` (created from a `v*` git tag).

### Run from source (Docker)

For development or customization, build the image yourself — a single image
(Go API + built UI) alongside PostgreSQL:

```bash
cd deploy
cp .env.example .env                                            # then edit secrets
docker compose run --rm --no-deps app hashpw 'your-password' > editor_hash.txt   # store editor hash
docker compose up -d --build
docker compose run --rm app seed                               # optional: load demo data
```

The app is then at `http://localhost:8080` (or `ATLAS_PORT`). It is publicly
**read-only** by default — log in as the editor to make changes or to toggle
public access off. All settings live in [`deploy/.env.example`](deploy/.env.example)
and [`server/README.md`](server/README.md).

### Frontend only (UI development)

**Prerequisites:** [Node.js](https://nodejs.org/) ≥ 18 and npm.

```bash
# 1. Clone
git clone https://github.com/tpasson/sw-atlas.git
cd sw-atlas

# 2. Install dependencies
npm install

# 3. Start the dev server (Vite, hot reload)
npm run dev
```

Then open the URL Vite prints (default `http://localhost:5173`).

| Command           | What it does                              |
| ----------------- | ----------------------------------------- |
| `npm run dev`     | Start the local dev server with HMR       |
| `npm run build`   | Build the production bundle to `dist/`     |
| `npm run preview` | Preview the production build locally       |
| `npm run build:demo` | Build the static, backend-less demo (GitHub Pages) |

## 📁 Project Structure

```
sw-atlas/
├─ index.html
├─ vite.config.js
├─ package.json
├─ Dockerfile               # multi-stage build: SPA + static Go binary
├─ src/                     # Vue 3 + Vite frontend
│  ├─ main.js               # app entry
│  ├─ App.vue               # root component & modal orchestration
│  ├─ api.js                # backend API client (demoApi.js for the Pages demo)
│  ├─ style.css
│  ├─ components/
│  │  ├─ TheHeader.vue      # toolbar: year nav, zoom, baselines, manage
│  │  ├─ MilestoneTable.vue # the timeline grid
│  │  ├─ MilestoneModal.vue # add/edit a milestone
│  │  └─ ManageModal.vue    # swimlanes, sub-lanes, settings
│  └─ stores/
│     └─ useAppStore.js     # reactive client state, synced with the API
├─ server/                  # Go backend (chi, pgx, goose)
│  ├─ cmd/atlas/            # entrypoint: serve · seed · hashpw
│  └─ internal/             # api · auth · store · db (migrations) · seed · config
├─ deploy/                  # docker-compose (+ ghcr) & .env example
└─ dist/                    # production build output (generated)
```

## 🗺️ Roadmap

ATLAS is delivered in phases — each phase is independently usable. The full plan
lives in [`ROADMAP.md`](ROADMAP.md).

- **P0 · Foundation** ✅ — Go backend + PostgreSQL + Docker, editor auth & public-read switch, frontend moved from `localStorage` to the API *(→ the real single source of truth)*
- **P1 · Visualization** ✅ — milestone vs. event types, time-spanning bars, marker shapes
- **P2 · Baselines** ✅ — named snapshots with switch & diff (added / moved / removed)
- **P3 · Export & reporting** — PPTX / image export of the timeline
- **P4 · Sync & drift** — generic project-plan import (CSV / Excel / ICS, optional API) & schedule-drift warnings

ATLAS stays a **general-purpose** milestone planner, not tied to any one domain.

## ⚙️ Configuration

ATLAS is configured entirely through environment variables (12-factor), so the
same image runs on a laptop, a LAN server, or corporate infrastructure. For
Docker, put them in `deploy/.env` (see [`deploy/.env.example`](deploy/.env.example));
the full list is documented in [`server/README.md`](server/README.md#configuration-environment-variables).

## 🤝 Contributing

Contributions are welcome! To propose a change:

1. **Open an issue** describing the bug or feature.
2. Fork the repo and create a branch (`git checkout -b feature/my-change`).
3. Commit with clear messages and open a **pull request** against `main`.

Please keep PRs focused and describe the user-facing impact. For larger features, start a discussion in an issue first so we can align on direction (see the [Roadmap](#%EF%B8%8F-roadmap)).

## 📄 License

Released under the **[Apache License 2.0](LICENSE)**. © 2026 Thomas Passon.
See [`NOTICE`](NOTICE) for attribution details.

## 🙏 Acknowledgements

Built with open source:

- [Go](https://go.dev/) — backend, with [chi](https://github.com/go-chi/chi), [pgx](https://github.com/jackc/pgx) and [goose](https://github.com/pressly/goose)
- [PostgreSQL](https://www.postgresql.org/) — database
- [Vue 3](https://vuejs.org/) + [Vite](https://vitejs.dev/) — frontend
- [Docker](https://www.docker.com/) — packaging & deployment
