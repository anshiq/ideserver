package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open connects via the pgx stdlib driver so sqlc's database/sql Queries
// work unchanged, then runs goose migrations up.
func Open(ctx context.Context, databaseURL string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		// Fall back to plain DSN passthrough.
		dbh, err2 := sql.Open("pgx", databaseURL)
		if err2 != nil {
			return nil, fmt.Errorf("db: open: %w", err)
		}
		if err := dbh.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("db: ping: %w", err)
		}
		if err := Migrate(dbh); err != nil {
			return nil, err
		}
		return dbh, nil
	}
	_ = cfg
	dbh := stdlib.OpenDB(*cfg)
	if err := dbh.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	if err := Migrate(dbh); err != nil {
		return nil, err
	}
	return dbh, nil
}

// Migrate runs embedded goose migrations up.
func Migrate(dbh *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.Up(dbh, "migrations"); err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	return nil
}
