// Package db handles the PostgreSQL connection pool and schema migrations.
// Migrations are embedded into the binary and run with goose; queries use pgxpool.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect opens a pgx connection pool and verifies connectivity.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func gooseDB(url string) (*sql.DB, error) {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

// Up applies all pending migrations.
func Up(url string) error {
	sqlDB, err := gooseDB(url)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return goose.Up(sqlDB, "migrations")
}

// Down rolls back the most recent migration.
func Down(url string) error {
	sqlDB, err := gooseDB(url)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return goose.Down(sqlDB, "migrations")
}
