<div align="center">

# 🛰️ ATLAS

**Collaborative, configurable planning — date-anchored timelines, custom artifact types, versioned history and change-request review.**

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Vue 3](https://img.shields.io/badge/Vue.js-3-42b883?logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![Vite](https://img.shields.io/badge/Vite-6-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](deploy/docker-compose.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#-contributing)
[![Status: Stable](https://img.shields.io/badge/status-stable-brightgreen.svg)](https://github.com/tpasson/sw-atlas/issues)
[![Release](https://img.shields.io/github/v/release/tpasson/sw-atlas)](https://github.com/tpasson/sw-atlas/releases)

[**🚀 Live Demo**](https://tpasson.github.io/sw-atlas/) · [Features](#-features) · [Getting Started](#-getting-started) · [Contributing](#-contributing)

</div>

---

## Overview

**ATLAS** is a visual planning tool that gives any team a single, shared picture of *what* happens *when* — across customers, events, products, platforms, features and releases.

Work is organized into **swimlanes** and **sub-lanes** on a month/year timeline. On that grid you place **milestones** (point-in-time markers) and **events** (bars that span a period), connect them with **typed relationships**, and drill into the *What / Why / Where / Who* behind each entry. Beyond the timeline you can define your **own item types** — Bug, Task, Release, anything — each with its own fields and behavior, browsed in an **Explorer** of folders by type.

ATLAS is **general-purpose** and not tied to any one domain: use the same workspace for product roadmaps, delivery plans, go-to-market calendars, release trains, backlogs or any program of work that benefits from a shared, configurable view.

> **Project status:** ATLAS is a collaborative, **multi-user** client–server application — a Vue SPA backed by a Go API and PostgreSQL. It supports **multi-member projects** (owner / editor / viewer roles), **configurable item types** with per-type fields, **item versioning & history**, **change requests** (propose a change, the owner approves it onto the plan), per-workspace public/private visibility, plan sharing & publishing between users, GitHub/Gitea source mirroring, and a self-hostable Docker deployment. See the [Releases](https://github.com/tpasson/sw-atlas/releases) page for the current version and changelog, and the [open issues](https://github.com/tpasson/sw-atlas/issues) for what's planned.

## ✨ Features

### Available today

**Plan modeling**
- 🧩 **Configurable item types** — define your own types (Bug, Task, Release, …) with per-type fields and a behavior family (timeline point / range, backlog item, folder); each type's icon & colour live on the type and flow into the legend automatically
- 🧭 **Milestones & events** with rich detail (*What / Why / Where*), **maturity** stages and **progress**; events span a period as bars
- 🔗 **Typed relationships** between items (depends-on, relates-to, child-of, implements, verifies), highlighted on hover, with late-dependency **risk warnings**
- 👤 **Assignees** — assign any item to a project member, shown as avatars

**Views**
- 📊 **Date-anchored timeline** — swimlanes & sub-lanes across fixed months and years, with a sticky header, lane packing, zoom and month/year navigation
- 🗂️ **Explorer** — browse every artifact in a two-pane **type tree + detail** pane, or as a **Table** or **Board** (kanban); add an item of any type inline
- 🧭 **Activity rail** — one click between Timeline / Explorer / Source Control / Change Requests; a configurable **facet filter** highlights items by type, assignee, maturity or group

**History & governance**
- 🕓 **Versioning & history** — every item carries a version and who created / last-edited it; each change is an immutable revision you can review and step back through
- 🧬 **Baselines** — save named snapshots and view the plan exactly as it stood then
- ✅ **Change requests** — members propose edits or new items; the owner approves them onto the plan (attributed to the proposer) or rejects them
- 🔐 **Roles** — owner / editor / viewer per project; workspace configuration (types, display, sources, sharing) is owner-only

**Multi-user & collaboration**
- 👥 **Accounts & projects** — an admin manages users; each user gets a personal workspace and can create **multi-member projects**, inviting others as editors or viewers
- 🔗 **Per-workspace URLs** — every plan lives at its own `/{username}` (or project) URL, public or private; viewers see it read-only
- 🧭 **Discovery landing page** — browse all public plans; admins can feature favourites
- 📡 **Share & publish** — publish a curated slice and let others subscribe to it, mirrored read-only and kept in sync — across separate ATLAS instances too

**Integrations**
- 🐙 **Source Control** — mirror a GitHub/Gitea repo's releases, tags, issues and pull requests as a read-only, browsable area; per-status colours; tokens encrypted at rest

**Hosting**
- 🗄️ **Backend + PostgreSQL** — a true single source of truth for the whole team
- 🐳 **Self-hostable** via Docker Compose (prebuilt multi-arch image), or run the browser-only build with local persistence

### Planned

Planned work is tracked entirely in **[GitHub issues](https://github.com/tpasson/sw-atlas/issues)** — for example export & reporting (PPTX / image) and generic project-plan import (CSV / Excel / ICS).

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
   ├─ Auth — accounts, multi-member projects (owner/editor/viewer), public/private
   ├─ Domain API (swimlanes · items · item types · typed relationships · baselines)
   ├─ History & governance (item revisions · change requests → approve/reject)
   ├─ Sharing & federation (publish scopes · subscribe & mirror, local or cross-instance)
   ├─ Repository sources ── GitHub / Gitea mirror (releases · tags · issues · PRs)
   └─ Export service (PPTX / image)                                          [planned]
   ▼
PostgreSQL  +  file storage (exports)
```

The collaborative stack — Vue SPA, Go API, PostgreSQL, multi-member projects,
configurable types, versioning & change requests, sharing/publishing and repository
mirroring — is live; the export service is still planned. The TLS-terminating reverse
proxy is provided by you in front of the container.

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

# 4. Create the editor login hash. The container runs as a non-root user, so the
#    hash file must be readable by it — chmod 644 (it's a one-way bcrypt hash).
docker run --rm ghcr.io/tpasson/sw-atlas:latest hashpw 'your-password' > editor_hash.txt
chmod 644 editor_hash.txt

# 5. Start it (pulls the image, runs app + PostgreSQL, migrates on startup)
docker compose -f docker-compose.ghcr.yml up -d

# 6. Optional: load the demo data
docker compose -f docker-compose.ghcr.yml run --rm app seed
```

ATLAS is then at `http://<host>:8080` (or `ATLAS_PORT`). The `ATLAS_EDITOR_USERNAME`
+ password you set bootstraps the **first admin** account; log in to edit, manage
users (admin → editor) and set each plan's visibility. New users each get their own
private workspace at `/{username}`, and `/` shows a directory of the public plans.

**Update later:**
```bash
docker compose -f docker-compose.ghcr.yml pull && docker compose -f docker-compose.ghcr.yml up -d
```

**Public deployment:** put a TLS-terminating reverse proxy (Caddy, Traefik, nginx)
in front — the container serves plain HTTP on its port. A specific version can be
pinned by replacing `:latest` with a release tag (e.g. `:vX.Y.Z`, created from a `v*` git tag).

### Run from source (Docker)

For development or customization, build the image yourself — a single image
(Go API + built UI) alongside PostgreSQL:

```bash
cd deploy
cp .env.example .env                                            # then edit secrets
docker compose run --rm --no-deps app hashpw 'your-password' > editor_hash.txt   # store editor hash
chmod 644 editor_hash.txt                                       # readable by the non-root container
docker compose up -d --build
docker compose run --rm app seed                               # optional: load demo data
```

The app is then at `http://localhost:8080` (or `ATLAS_PORT`). Log in with the
bootstrapped admin account to edit and manage users. All settings live in
[`deploy/.env.example`](deploy/.env.example) and [`server/README.md`](server/README.md).

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
│  │  ├─ ActivityRail.vue        # left rail: views · theme · account
│  │  ├─ TheHeader.vue           # toolbar: date nav · zoom · baselines · project switcher
│  │  ├─ LandingPage.vue         # discovery directory of public plans (the "/" home)
│  │  ├─ MilestoneTable.vue      # the timeline grid
│  │  ├─ ExplorerView.vue        # type tree + detail · Table · Board
│  │  ├─ SourceControlView.vue   # mirrored Git sources & content
│  │  ├─ ChangeRequestsView.vue  # propose / approve change requests
│  │  ├─ MilestoneModal.vue      # add/edit an item (+ History tab · propose)
│  │  ├─ TypesManager.vue        # configurable item types & fields
│  │  └─ ManageModal.vue         # areas · display · types · baselines · sharing · members · users
│  └─ stores/
│     └─ useAppStore.js     # reactive client state, synced with the API
├─ server/                  # Go backend (chi, pgx, goose)
│  ├─ cmd/atlas/            # entrypoint: serve · seed · hashpw
│  └─ internal/             # api · auth · store · db (migrations) · seed · config
├─ deploy/                  # docker-compose (+ ghcr) & .env example
└─ dist/                    # production build output (generated)
```

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

Please keep PRs focused and describe the user-facing impact. For larger features, start a discussion in an issue first so we can align on direction.

## 📄 License

Released under the **[Apache License 2.0](LICENSE)**. © 2026 Thomas Passon.
See [`NOTICE`](NOTICE) for attribution details.

## 🙏 Acknowledgements

Built with open source:

- [Go](https://go.dev/) — backend, with [chi](https://github.com/go-chi/chi), [pgx](https://github.com/jackc/pgx) and [goose](https://github.com/pressly/goose)
- [PostgreSQL](https://www.postgresql.org/) — database
- [Vue 3](https://vuejs.org/) + [Vite](https://vitejs.dev/) — frontend
- [Docker](https://www.docker.com/) — packaging & deployment
