package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/logging"
)

func (s *Storage) Migrate(ctx context.Context) error {
	currentVersion, err := s.schemaVersion(ctx)
	if err != nil {
		return err
	}

	log := logging.FromContext(ctx)
	log.Info("Running database migrations",
		slog.Int("current_version", currentVersion),
		slog.Int("latest_version", schemaVersion))

	if currentVersion == 0 {
		log.Info("Create initial database schema")
		if err := s.applyVersion(ctx, currentVersion); err != nil {
			return err
		}
		currentVersion, err = s.schemaVersion(ctx)
		if err != nil {
			return err
		}
		if currentVersion < schemaVersion {
			log.Info("Apply next migrations",
				slog.Int("current_version", currentVersion))
		}
	}

	for i := currentVersion; i < schemaVersion; i++ {
		if err := s.applyVersion(ctx, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) schemaVersion(ctx context.Context) (int, error) {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (
  SELECT FROM pg_tables WHERE tablename = 'schema_version')`)

	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return 0, fmt.Errorf("storage: looking for schema_version table: %w", err)
	} else if !exists {
		return 0, nil
	}

	rows, _ = s.db.Query(ctx,
		`SELECT CAST(version AS INTEGER) FROM schema_version`)
	ver, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("storage: unable fetch schema version: %w", err)
	}
	return ver, nil
}

func (s *Storage) applyVersion(ctx context.Context, v int) error {
	nextVersion := v + 1
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if err := migrations[v].Do(ctx, tx); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
		if v > 0 {
			_, err := tx.Exec(ctx, `UPDATE schema_version SET version = $1`,
				strconv.FormatInt(int64(nextVersion), 10))
			if err != nil {
				return fmt.Errorf("update version: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("migration %d -> %d: %w", v, nextVersion, err)
	}
	return nil
}

// IsSchemaUpToDate checks if the database schema is up to date.
func (s *Storage) SchemaUpToDate(ctx context.Context) error {
	currentVersion, err := s.schemaVersion(ctx)
	if err != nil {
		return err
	}

	if currentVersion < schemaVersion {
		return fmt.Errorf(
			`storage: the database schema is not up to date: current=v%d expected=v%d`,
			currentVersion, schemaVersion)
	}
	return nil
}
