// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package database // import "miniflux.app/v2/internal/database"

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
)

// Migrate executes database migrations.
func Migrate(db *sql.DB) error {
	var currentVersion int
	err := db.QueryRow(`SELECT version FROM schema_version`).
		Scan(&currentVersion)
	if err != nil {
		if err = undefinedTable(err); err != nil {
			return fmt.Errorf("database: failed select version: %w", err)
		}
	}

	driver := getDriverStr()
	slog.Info("Running database migrations",
		slog.Int("current_version", currentVersion),
		slog.Int("latest_version", schemaVersion),
		slog.String("driver", driver),
	)

	for version := currentVersion; version < schemaVersion; version++ {
		newVersion := version + 1

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("[Migration v%d] %w", newVersion, err)
		}

		if err := migrations[version](tx, driver); err != nil {
			tx.Rollback()
			return fmt.Errorf("[Migration v%d] %w", newVersion, err)
		}

		if _, err := tx.Exec(`DELETE FROM schema_version`); err != nil {
			tx.Rollback()
			return fmt.Errorf("[Migration v%d] %w", newVersion, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES ($1)`, newVersion); err != nil {
			tx.Rollback()
			return fmt.Errorf("[Migration v%d] %w", newVersion, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("[Migration v%d] %w", newVersion, err)
		}
	}

	return nil
}

func undefinedTable(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code.Name() == "undefined_table" {
			return nil
		}
		return fmt.Errorf("%s: %w", pqErr.Code.Name(), err)
	}
	return err
}

// IsSchemaUpToDate checks if the database schema is up to date.
func IsSchemaUpToDate(db *sql.DB) error {
	var currentVersion int
	err := db.QueryRow(`SELECT version FROM schema_version`).
		Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("database: failed select version: %w", err)
	}
	if currentVersion < schemaVersion {
		return fmt.Errorf(`the database schema is not up to date: current=v%d expected=v%d`, currentVersion, schemaVersion)
	}
	return nil
}
