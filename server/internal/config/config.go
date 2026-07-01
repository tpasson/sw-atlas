// Package config loads runtime configuration from environment variables
// (12-factor), so the same binary runs locally, in Docker, and in corporate infra.
package config

import (
	"os"
	"strings"
)

type Config struct {
	ListenAddr     string
	DatabaseURL    string
	SessionSecret  string
	EditorUsername string
	EditorHash     string // bcrypt hash of the editor password
	StaticDir      string // optional: directory with the built frontend to serve
}

func Load() Config {
	return Config{
		ListenAddr:     env("ATLAS_LISTEN_ADDR", ":8080"),
		DatabaseURL:    env("ATLAS_DATABASE_URL", "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable"),
		SessionSecret:  env("ATLAS_SESSION_SECRET", "dev-insecure-change-me"),
		EditorUsername: env("ATLAS_ADMIN_USERNAME", env("ATLAS_EDITOR_USERNAME", "atlas-admin")),
		EditorHash:     editorHash(),
		StaticDir:      os.Getenv("ATLAS_STATIC_DIR"),
	}
}

// editorHash reads the bcrypt hash from ATLAS_EDITOR_PASSWORD_HASH, or, if that
// is empty, from the file named by ATLAS_EDITOR_PASSWORD_HASH_FILE. The file
// variant avoids passing a '$'-laden hash through docker-compose interpolation
// (use a Docker secret).
func editorHash() string {
	if h := os.Getenv("ATLAS_EDITOR_PASSWORD_HASH"); h != "" {
		return h
	}
	if f := os.Getenv("ATLAS_EDITOR_PASSWORD_HASH_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
