package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenPostgres(ctx context.Context, connectionURL string) (*sql.DB, error) {
	if strings.TrimSpace(connectionURL) == "" {
		return nil, fmt.Errorf("WATCHLIST_DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize PostgreSQL schema: %w", err)
	}
	return db, nil
}
