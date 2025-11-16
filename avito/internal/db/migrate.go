package db

import (
	"context"
	"database/sql"
	"embed"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	sqlBytes, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, string(sqlBytes))
	return err
}
