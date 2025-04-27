// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"miniflux.app/v2/internal/logging"
)

// New returns a new Storage.
func New(ctx context.Context, connString string, maxConns, minConns int,
	lifeTime time.Duration,
) (*Storage, error) {
	c, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("storage: parse connection string: %w", err)
	}

	c.MaxConns = int32(maxConns)
	c.MinConns = int32(minConns)
	c.MaxConnLifetime = lifeTime

	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("storage: new pgx pool: %w", err)
	}
	return &Storage{db: p}, nil
}

// Storage handles all operations related to the database.
type Storage struct {
	db *pgxpool.Pool
}

func (s *Storage) Close(ctx context.Context) {
	stat := s.db.Stat()
	logging.FromContext(ctx).Info("pgx pool statistics",
		slog.Group("acquire",
			slog.Int64("count", stat.AcquireCount()),
			slog.Duration("duration", stat.AcquireDuration()),
			slog.Int64("conns", int64(stat.AcquiredConns())),
			slog.Int64("canceled_count", stat.CanceledAcquireCount()),
			slog.Int64("empty_count", stat.EmptyAcquireCount()),
			slog.Duration("empty_wait_time", stat.EmptyAcquireWaitTime()),
		),
		slog.Int64("constructing_conns", int64(stat.ConstructingConns())),
		slog.Int64("idle_conns", int64(stat.IdleConns())),
		slog.Int64("max_conns", int64(stat.MaxConns())),
		slog.Int64("max_idle_destroy_count", stat.MaxIdleDestroyCount()),
		slog.Int64("max_lifetime_destroy_count", stat.MaxLifetimeDestroyCount()),
		slog.Int64("new_conns_count", stat.NewConnsCount()),
		slog.Int64("total_conns", int64(stat.TotalConns())),
	)
	s.db.Close()
}

// DatabaseVersion returns the version of the database which is in use.
func (s *Storage) DatabaseVersion(ctx context.Context) string {
	rows, _ := s.db.Query(ctx,
		`SELECT current_setting('server_version')`)
	dbVersion, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
	if err != nil {
		return err.Error()
	}
	return dbVersion
}

// Ping checks if the database connection works.
func (s *Storage) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		return fmt.Errorf("storage: ping failed: %w", err)
	}
	return nil
}

// DBSize returns how much size the database is using in a pretty way.
func (s *Storage) DBSize(ctx context.Context) (string, error) {
	rows, _ := s.db.Query(ctx,
		"SELECT pg_size_pretty(pg_database_size(current_database()))")
	size, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
	if err != nil {
		return "", fmt.Errorf("storage: %w", err)
	}
	return size, nil
}
