// Command atlas is the ATLAS backend. Without arguments it serves the API
// (and the built frontend if ATLAS_STATIC_DIR is set). Subcommands:
//
//	atlas serve         start the HTTP server (default)
//	atlas migrate       apply pending DB migrations
//	atlas migrate-down  roll back the most recent migration
//	atlas seed          load demo data (idempotent)
//	atlas hashpw <pw>   print a bcrypt hash for ATLAS_EDITOR_PASSWORD_HASH
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tpasson/sw-atlas/server/internal/api"
	"github.com/tpasson/sw-atlas/server/internal/auth"
	"github.com/tpasson/sw-atlas/server/internal/config"
	"github.com/tpasson/sw-atlas/server/internal/db"
	"github.com/tpasson/sw-atlas/server/internal/seed"
	"github.com/tpasson/sw-atlas/server/internal/store"
)

func main() {
	cfg := config.Load()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "serve":
		runServe(cfg)
	case "migrate":
		must(db.Up(cfg.DatabaseURL))
		fmt.Println("migrations applied")
	case "migrate-down":
		must(db.Down(cfg.DatabaseURL))
		fmt.Println("rolled back one migration")
	case "seed":
		runSeed(cfg)
	case "hashpw":
		runHashpw()
	default:
		fmt.Printf("unknown command %q (use: serve | migrate | migrate-down | seed | hashpw)\n", cmd)
		os.Exit(2)
	}
}

func runServe(cfg config.Config) {
	must(db.Up(cfg.DatabaseURL)) // auto-migrate on startup

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	must(err)
	defer pool.Close()

	st := store.New(pool)
	au := auth.New(cfg.SessionSecret)
	handler := api.NewRouter(st, au, cfg.StaticDir)

	// Bootstrap the first admin from the env editor credentials (no-op once any
	// account exists). Accounts are managed in-app thereafter.
	must(st.EnsureBootstrapAdmin(context.Background(), cfg.EditorUsername, cfg.EditorHash))
	if n, err := st.CountUsers(context.Background()); err == nil && n == 0 {
		log.Println("WARNING: no user accounts exist and ATLAS_EDITOR_PASSWORD_HASH is empty — login is disabled until you bootstrap an admin (set ATLAS_EDITOR_USERNAME + ATLAS_EDITOR_PASSWORD_HASH, see: atlas hashpw <password>)")
	}

	// Background smart-poll: sync due subscriptions + GitHub sources every minute.
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for range t.C {
			st.SyncDueSubscriptions(context.Background())
			st.SyncDueGitHubSources(context.Background())
		}
	}()

	log.Printf("ATLAS listening on %s", cfg.ListenAddr)
	must(http.ListenAndServe(cfg.ListenAddr, handler))
}

func runSeed(cfg config.Config) {
	must(db.Up(cfg.DatabaseURL))
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	must(err)
	defer pool.Close()

	n, err := seed.Run(context.Background(), store.New(pool), store.DefaultWorkspaceID)
	must(err)
	fmt.Printf("seed complete (%d swimlanes present)\n", n)
}

func runHashpw() {
	if len(os.Args) < 3 {
		fmt.Println("usage: atlas hashpw <password>")
		os.Exit(2)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(os.Args[2]), bcrypt.DefaultCost)
	must(err)
	fmt.Println(string(h))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
