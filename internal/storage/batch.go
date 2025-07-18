// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"miniflux.app/v2/internal/model"
)

type BatchBuilder struct {
	db         *pgxpool.Pool
	args       []any
	conditions []string
	limit      int
}

func (s *Storage) NewBatchBuilder() *BatchBuilder {
	return &BatchBuilder{db: s.db}
}

func (b *BatchBuilder) WithBatchSize(batchSize int) *BatchBuilder {
	b.limit = batchSize
	return b
}

func (b *BatchBuilder) WithUserID(userID int64) *BatchBuilder {
	b.conditions = append(b.conditions,
		fmt.Sprintf("user_id = $%d", len(b.args)+1))
	b.args = append(b.args, userID)
	return b
}

func (b *BatchBuilder) WithCategoryID(categoryID int64) *BatchBuilder {
	b.conditions = append(b.conditions,
		fmt.Sprintf("category_id = $%d", len(b.args)+1))
	b.args = append(b.args, categoryID)
	return b
}

func (b *BatchBuilder) WithErrorLimit(limit int) *BatchBuilder {
	if limit > 0 {
		b.conditions = append(b.conditions,
			fmt.Sprintf("parsing_error_count < $%d", len(b.args)+1))
		b.args = append(b.args, limit)
	}
	return b
}

func (b *BatchBuilder) WithNextCheckExpired() *BatchBuilder {
	b.conditions = append(b.conditions, "next_check_at <= now()")
	return b
}

func (b *BatchBuilder) WithoutDisabledFeeds() *BatchBuilder {
	b.conditions = append(b.conditions, "disabled is false")
	return b
}

func (b *BatchBuilder) FetchJobs(ctx context.Context) ([]model.Job, error) {
	query := "SELECT id, user_id, feed_url FROM feeds"
	if len(b.conditions) > 0 {
		query += " WHERE " + strings.Join(b.conditions, " AND ")
	}

	if b.limit > 0 {
		query += fmt.Sprintf(" ORDER BY next_check_at ASC LIMIT %d", b.limit)
	}

	rows, _ := b.db.Query(ctx, query, b.args...)
	jobs, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Job])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch batch of jobs: %w`, err)
	}
	return jobs, nil
}

func (b *BatchBuilder) ResetNextCheckAt(ctx context.Context) error {
	query := `
UPDATE feeds SET
  parsing_error_count = 0,
  parsing_error_msg		= '',
	next_check_at				= now()`

	if len(b.conditions) > 0 {
		query += " WHERE " + strings.Join(b.conditions, " AND ")
	}

	if _, err := b.db.Exec(ctx, query, b.args...); err != nil {
		return fmt.Errorf("storage: failed reset next check: %w", err)
	}
	return nil
}
