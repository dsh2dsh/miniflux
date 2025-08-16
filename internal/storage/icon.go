// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// HasFeedIcon checks if the given feed has an icon.
func (s *Storage) HasFeedIcon(ctx context.Context, feedID int64) bool {
	rows, _ := s.db.Query(ctx,
		`SELECT EXISTS(SELECT FROM feed_icons WHERE feed_id = $1)`, feedID)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("storage: unable check has feed icon",
			slog.Int64("feed_id", feedID), slog.Any("error", err))
		return false
	}
	return result
}

// IconByID returns an icon by the ID.
func (s *Storage) IconByID(ctx context.Context, iconID int64,
) (*model.Icon, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT id, hash, mime_type, content FROM icons WHERE id = $1`, iconID)

	icon, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Icon])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("store: unable to fetch icon #%d: %w", iconID, err)
	}
	return icon, nil
}

// IconByExternalID returns an icon by the External Icon ID.
func (s *Storage) IconByExternalID(ctx context.Context, externalIconID string,
) (*model.Icon, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT id, hash, mime_type, content FROM icons WHERE hash = $1`,
		externalIconID)

	icon, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Icon])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("store: unable to fetch icon %s: %w",
			externalIconID, err)
	}
	return icon, nil
}

// IconByFeedID returns a feed icon.
func (s *Storage) IconByFeedID(ctx context.Context, userID, feedID int64,
) (*model.Icon, error) {
	rows, _ := s.db.Query(ctx, `
SELECT icons.id, icons.hash, icons.mime_type, icons.content
  FROM feeds, feed_icons, icons
 WHERE feeds.id = $1 AND feeds.user_id = $2
       AND feeds.id = feed_icons.feed_id
       AND feed_icons.icon_id = icons.id`, feedID, userID)

	icon, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Icon])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch icon: %w`, err)
	}
	return icon, nil
}

// StoreFeedIcon creates or updates a feed icon.
func (s *Storage) StoreFeedIcon(ctx context.Context, feedID int64,
	icon *model.Icon,
) error {
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, _ := tx.Query(ctx, `
INSERT INTO icons (hash, mime_type, content)
           VALUES ($1,   $2,        $3)
ON CONFLICT (hash)
  DO UPDATE SET mime_type = $2, content = $3
RETURNING id`, icon.Hash, normalizeMimeType(icon.MimeType), icon.Content)

		id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64])
		if err != nil {
			return fmt.Errorf("create feed icon: %w", err)
		}
		icon.ID = id

		_, err = tx.Exec(ctx, `
INSERT INTO feed_icons (feed_id, icon_id) VALUES ($1, $2)
ON CONFLICT (feed_id)
  DO UPDATE SET icon_id = $2`, feedID, icon.ID)
		if err != nil {
			return fmt.Errorf("associate feed and icon: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("storage: unable to store feed icon: %w", err)
	}
	return nil
}

func normalizeMimeType(mimeType string) string {
	mimeType = strings.ToLower(mimeType)
	switch mimeType {
	case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/svg+xml", "image/gif":
		return mimeType
	}
	return "image/x-icon"
}

// Icons returns all icons that belongs to a user.
func (s *Storage) Icons(ctx context.Context, userID int64,
) ([]*model.Icon, error) {
	rows, _ := s.db.Query(ctx, `
SELECT icons.id, icons.hash, icons.mime_type, icons.content
  FROM feeds, feed_icons, icons
 WHERE feeds.user_id = $1
       AND feeds.id = feed_icons.feed_id
       AND feed_icons.icon_id = icons.id`, userID)

	icons, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[model.Icon])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch icons: %w`, err)
	}
	return icons, nil
}

func (s *Storage) DeleteLostIcons(ctx context.Context) (int64, error) {
	result, err := s.db.Exec(ctx, `
WITH lost_icons AS (
  SELECT i.id
    FROM icons i LEFT JOIN feed_icons fi ON (fi.icon_id = i.id)
   WHERE fi.feed_id IS NULL
)
DELETE FROM icons AS i
 USING lost_icons AS li
 WHERE i.id = li.id`)
	if err != nil {
		return 0, fmt.Errorf("storage: delete lost icons: %w", err)
	}
	return result.RowsAffected(), nil
}
