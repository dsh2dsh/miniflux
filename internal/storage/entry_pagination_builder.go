// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"miniflux.app/v2/internal/model"
)

// NewEntryPaginationBuilder returns a new EntryPaginationBuilder.
func (s *Storage) NewEntryPaginationBuilder(userID, entryID int64, order,
	direction string,
) *EntryPaginationBuilder {
	return &EntryPaginationBuilder{
		db:         s.db,
		args:       []any{userID, "removed"},
		conditions: []string{"e.user_id = $1", "e.status <> $2"},
		entryID:    entryID,
		order:      order,
		direction:  direction,
	}
}

// EntryPaginationBuilder is a builder for entry prev/next queries.
type EntryPaginationBuilder struct {
	db         *pgxpool.Pool
	conditions []string
	args       []any
	entryID    int64
	order      string
	direction  string
}

// WithSearchQuery adds full-text search query to the condition.
func (e *EntryPaginationBuilder) WithSearchQuery(query string,
) *EntryPaginationBuilder {
	if query != "" {
		e.conditions = append(e.conditions, fmt.Sprintf(
			"e.document_vectors @@ websearch_to_tsquery($%d)",
			len(e.args)+1))
		e.args = append(e.args, query)
	}
	return e
}

// WithStarred adds starred to the condition.
func (e *EntryPaginationBuilder) WithStarred() *EntryPaginationBuilder {
	e.conditions = append(e.conditions, "e.starred is true")
	return e
}

// WithFeedID adds feed_id to the condition.
func (e *EntryPaginationBuilder) WithFeedID(feedID int64,
) *EntryPaginationBuilder {
	if feedID != 0 {
		e.conditions = append(e.conditions,
			fmt.Sprintf("e.feed_id = $%d", len(e.args)+1))
		e.args = append(e.args, feedID)
	}
	return e
}

// WithCategoryID adds category_id to the condition.
func (e *EntryPaginationBuilder) WithCategoryID(categoryID int64,
) *EntryPaginationBuilder {
	if categoryID != 0 {
		e.conditions = append(e.conditions,
			fmt.Sprintf("f.category_id = $%d", len(e.args)+1))
		e.args = append(e.args, categoryID)
	}
	return e
}

// WithStatus adds status to the condition.
func (e *EntryPaginationBuilder) WithStatus(status string,
) *EntryPaginationBuilder {
	if status != "" {
		e.conditions = append(e.conditions,
			fmt.Sprintf("e.status = $%d", len(e.args)+1))
		e.args = append(e.args, status)
	}
	return e
}

func (e *EntryPaginationBuilder) WithTags(tags []string,
) *EntryPaginationBuilder {
	if len(tags) > 0 {
		for _, tag := range tags {
			e.conditions = append(e.conditions,
				fmt.Sprintf("LOWER($%d) = ANY(LOWER(e.tags::text)::text[])",
					len(e.args)+1))
			e.args = append(e.args, tag)
		}
	}
	return e
}

// WithGloballyVisible adds global visibility to the condition.
func (e *EntryPaginationBuilder) WithGloballyVisible() *EntryPaginationBuilder {
	e.conditions = append(e.conditions, "not c.hide_globally")
	e.conditions = append(e.conditions, "not f.hide_globally")
	return e
}

// Entries returns previous and next entries.
func (e *EntryPaginationBuilder) Entries(ctx context.Context) (prevEntry,
	nextEntry *model.Entry, _ error,
) {
	err := pgx.BeginFunc(ctx, e.db, func(tx pgx.Tx) error {
		prevID, nextID, err := e.getPrevNextID(ctx, tx)
		if err != nil {
			return err
		}

		prevEntry, err = e.getEntry(ctx, tx, prevID)
		if err != nil {
			return err
		}

		nextEntry, err = e.getEntry(ctx, tx, nextID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf(
			"store: unable fetch entries for pagination: %w", err)
	}

	if e.direction == "desc" {
		return nextEntry, prevEntry, nil
	}
	return prevEntry, nextEntry, nil
}

func (e *EntryPaginationBuilder) getPrevNextID(ctx context.Context, tx pgx.Tx,
) (int64, int64, error) {
	cte := `
WITH entry_pagination AS (
  SELECT e.id,
	       lag(e.id) over (order by e.%[1]s asc, e.created_at asc, e.id desc) as prev_id,
	       lead(e.id) over (order by e.%[1]s asc, e.created_at asc, e.id desc) as next_id
    FROM entries AS e
         JOIN feeds AS f ON f.id=e.feed_id
         JOIN categories c ON c.id = f.category_id
   WHERE %[2]s
   ORDER BY e.%[1]s asc, e.created_at asc, e.id desc
)
SELECT prev_id, next_id FROM entry_pagination AS ep WHERE %[3]s`

	subCondition := strings.Join(e.conditions, " AND ")
	finalCondition := fmt.Sprintf("ep.id = $%d", len(e.args)+1)
	query := fmt.Sprintf(cte, e.order, subCondition, finalCondition)
	e.args = append(e.args, e.entryID)

	var pID, nID pgtype.Int8
	err := tx.QueryRow(ctx, query, e.args...).Scan(&pID, &nID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, fmt.Errorf("entry pagination: %w", err)
	}

	var prevID, nextID int64
	if pID.Valid {
		prevID = pID.Int64
	}
	if nID.Valid {
		nextID = nID.Int64
	}
	return prevID, nextID, nil
}

func (e *EntryPaginationBuilder) getEntry(ctx context.Context, tx pgx.Tx,
	entryID int64,
) (*model.Entry, error) {
	rows, _ := tx.Query(ctx,
		`SELECT id, title FROM entries WHERE id = $1`, entryID)

	entry, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.Entry])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("fetching sibling entry: %w", err)
	}
	return entry, nil
}
