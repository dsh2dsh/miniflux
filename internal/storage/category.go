// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// AnotherCategoryExists checks if another category exists with the same title.
func (s *Storage) AnotherCategoryExists(ctx context.Context, userID,
	categoryID int64, title string,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (
  SELECT FROM categories
  WHERE user_id=$1 AND id != $2 AND lower(title)=lower($3) LIMIT 1)`,
		userID, categoryID, title)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("failed category lookup",
			slog.Int64("user_id", userID),
			slog.Int64("category_id", categoryID),
			slog.String("title", title),
			slog.Any("error", err))
		return false
	}
	return result
}

// CategoryTitleExists checks if the given category exists into the database.
func (s *Storage) CategoryTitleExists(ctx context.Context, userID int64,
	title string,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (
  SELECT FROM categories
  WHERE user_id=$1 AND lower(title)=lower($2) LIMIT 1)`,
		userID, title)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("failed category lookup",
			slog.Int64("user_id", userID),
			slog.String("title", title),
			slog.Any("error", err))
		return false
	}
	return result
}

// CategoryIDExists checks if the given category exists into the database.
func (s *Storage) CategoryIDExists(ctx context.Context, userID,
	categoryID int64,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (SELECT FROM categories WHERE user_id=$1 AND id=$2)`,
		userID, categoryID)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("failed category lookup",
			slog.Int64("user_id", userID),
			slog.Int64("category_id", categoryID),
			slog.Any("error", err))
		return false
	}
	return result
}

// Category returns a category from the database.
func (s *Storage) Category(ctx context.Context, userID, categoryID int64,
) (*model.Category, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, title, hide_globally
  FROM categories
 WHERE user_id=$1 AND id=$2`,
		userID, categoryID)

	category, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.Category])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch category: %w`, err)
	}
	return category, nil
}

// FirstCategory returns the first category for the given user.
func (s *Storage) FirstCategory(ctx context.Context, userID int64,
) (*model.Category, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, title, hide_globally
  FROM categories
 WHERE user_id=$1
 ORDER BY title ASC LIMIT 1`,
		userID)

	category, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.Category])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch category: %w`, err)
	}
	return category, nil
}

// CategoryByTitle finds a category by the title.
func (s *Storage) CategoryByTitle(ctx context.Context, userID int64,
	title string,
) (*model.Category, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, title, hide_globally
  FROM categories
 WHERE user_id=$1 AND title=$2`,
		userID, title)

	category, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.Category])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch category: %w`, err)
	}
	return category, nil
}

// Categories returns all categories that belongs to the given user.
func (s *Storage) Categories(ctx context.Context, userID int64,
) ([]*model.Category, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, title, hide_globally
  FROM categories
 WHERE user_id=$1 ORDER BY title ASC`,
		userID)

	categories, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByNameLax[model.Category])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch category row: %w`, err)
	}
	return categories, nil
}

// CategoriesWithFeedCount returns all categories with the number of feeds.
func (s *Storage) CategoriesWithFeedCount(ctx context.Context, userID int64,
) ([]*model.Category, error) {
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
SELECT c.id, c.user_id, c.title, c.hide_globally,
	     (SELECT count(*) FROM feeds WHERE feeds.category_id=c.id) AS feed_count,
	     (SELECT count(*)
		      FROM feeds
			    JOIN entries ON (feeds.id = entries.feed_id)
		     WHERE feeds.category_id = c.id AND entries.status = $1) AS total_unread
  FROM categories c
 WHERE user_id=$2`

	if user.CategoriesSortingOrder == "alphabetical" {
		query += ` ORDER BY c.title ASC`
	} else {
		query += ` ORDER BY total_unread DESC, c.title ASC`
	}

	rows, _ := s.db.Query(ctx, query, model.EntryStatusUnread, userID)
	categories, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.Category])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch categories: %w`, err)
	}
	return categories, nil
}

// CreateCategory creates a new category.
func (s *Storage) CreateCategory(ctx context.Context, userID int64,
	request *model.CategoryCreationRequest,
) (*model.Category, error) {
	rows, _ := s.db.Query(ctx, `
INSERT INTO categories (user_id, title, hide_globally)
                VALUES ($1,      $2,    $3)
  RETURNING id, user_id, title, hide_globally`,
		userID, request.Title, request.HideGlobally)

	category, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.Category])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to create category %q: %w`,
			request.Title, err)
	}
	return category, nil
}

// UpdateCategory updates an existing category.
func (s *Storage) UpdateCategory(ctx context.Context, category *model.Category,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE categories
   SET title=$1, hide_globally=$2
 WHERE id=$3 AND user_id=$4`,
		category.Title, category.HideGlobally, category.ID, category.UserID)
	if err != nil {
		return fmt.Errorf(`store: unable to update category: %w`, err)
	}
	return nil
}

// RemoveCategory deletes a category.
func (s *Storage) RemoveCategory(ctx context.Context, userID, categoryID int64,
) error {
	result, err := s.db.Exec(ctx,
		`DELETE FROM categories WHERE id = $1 AND user_id = $2`,
		categoryID, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to remove this category: %w`, err)
	}

	if result.RowsAffected() == 0 {
		return errors.New(`store: no category has been removed`)
	}
	return nil
}

// delete the given categories, replacing those categories with the user's first
// category on affected feeds
func (s *Storage) RemoveAndReplaceCategoriesByName(ctx context.Context,
	userid int64, titles []string,
) error {
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, _ := tx.Query(ctx, `
SELECT count(*) FROM categories WHERE user_id = $1 and title != ANY($2)`,
			userid, titles)
		count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
		if err != nil {
			return fmt.Errorf("retrieve category count: %w", err)
		} else if count < 1 {
			return errors.New("at least 1 category must remain after deletion")
		}

		_, err = tx.Exec(ctx, `
WITH d_cats AS (
  SELECT id FROM categories WHERE user_id = $1 AND title = ANY($2))
UPDATE feeds SET category_id = (
  SELECT id
	  FROM categories
	 WHERE user_id = $1 AND id NOT IN (SELECT id FROM d_cats)
	 ORDER BY title ASC LIMIT 1)
 WHERE user_id = $1 AND category_id IN (SELECT id FROM d_cats)`,
			userid, titles)
		if err != nil {
			return fmt.Errorf("update categories: %w", err)
		}

		_, err = tx.Exec(ctx,
			`DELETE FROM categories WHERE user_id = $1 AND title = ANY($2)`,
			userid, titles)
		if err != nil {
			return fmt.Errorf("delete categories: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("store: unable to replace categories: %w", err)
	}
	return nil
}
