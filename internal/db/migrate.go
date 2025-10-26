package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/test-task-front/wms/migrations"
)

func RunMigrations(ctx context.Context, database *sqlx.DB) error {
	if _, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrationFS := migrations.FS()
	files, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(file.Name(), ".sql")

		var exists bool
		if err := database.GetContext(ctx, &exists, `
			SELECT EXISTS (
				SELECT 1 FROM schema_migrations WHERE version = $1
			)
		`, version); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}

		content, err := fs.ReadFile(migrationFS, file.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		upSQL := extractUp(string(content))
		if strings.TrimSpace(upSQL) == "" {
			continue
		}

		tx, err := database.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, upSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations (version) VALUES ($1)
		`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}

	return nil
}

func extractUp(content string) string {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"

	if idx := strings.Index(content, upMarker); idx >= 0 {
		content = content[idx+len(upMarker):]
	}
	if idx := strings.Index(content, downMarker); idx >= 0 {
		content = content[:idx]
	}
	return content
}
