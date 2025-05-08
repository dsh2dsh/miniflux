// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/model"
)

// GetEnclosures returns all attachments for the given entry.
func (s *Storage) GetEnclosures(ctx context.Context, entryID int64,
) (model.EnclosureList, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, entry_id, url, size, mime_type, media_progression
  FROM enclosures
WHERE entry_id = $1 ORDER BY id ASC`,
		entryID)

	enclosures, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.Enclosure])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch enclosures: %w`, err)
	}
	return enclosures, nil
}

func (s *Storage) GetEnclosuresForEntries(ctx context.Context, entryIDs []int64,
) (map[int64]model.EnclosureList, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, entry_id, url, size, mime_type, media_progression
  FROM enclosures
WHERE entry_id = ANY($1) ORDER BY id ASC`,
		entryIDs)

	enclosures, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.Enclosure])
	if err != nil {
		return nil, fmt.Errorf("store: unable to fetch enclosures: %w", err)
	}

	enclosuresMap := make(map[int64]model.EnclosureList)
	for _, enclosure := range enclosures {
		enclosuresMap[enclosure.EntryID] = append(
			enclosuresMap[enclosure.EntryID], enclosure)
	}
	return enclosuresMap, nil
}

func (s *Storage) GetEnclosure(ctx context.Context, enclosureID int64,
) (*model.Enclosure, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, entry_id, url, size, mime_type, media_progression
  FROM enclosures
 WHERE id = $1 ORDER BY id ASC`,
		enclosureID)

	enclosure, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Enclosure])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch enclosure row: %w`, err)
	}
	return enclosure, nil
}

func (s *Storage) createEnclosures(ctx context.Context, tx pgx.Tx,
	enclosures model.EnclosureList,
) error {
	enclosures, mapped := enclosures.Uniq()
	switch len(enclosures) {
	case 0:
		return nil
	case 1:
		return s.createEnclosure(ctx, tx, enclosures[0])
	}

	_, err := tx.CopyFrom(ctx, pgx.Identifier{"enclosures"},
		[]string{
			"url",
			"size",
			"mime_type",
			"entry_id",
			"user_id",
			"media_progression",
		},
		pgx.CopyFromSlice(len(enclosures), func(i int) ([]any, error) {
			e := enclosures[i]
			return []any{
				e.URL,
				e.Size,
				e.MimeType,
				e.EntryID,
				e.UserID,
				e.MediaProgression,
			}, nil
		}))
	if err != nil {
		return fmt.Errorf("storage: copy from enclosures: %w", err)
	}

	rows, _ := tx.Query(ctx,
		`SELECT id, entry_id, url FROM enclosures WHERE entry_id = ANY($1)`,
		slices.Collect(maps.Keys(mapped)))

	var id, entryID int64
	var url string

	_, err = pgx.ForEachRow(rows, []any{&id, &entryID, &url},
		func() error {
			if byURL, ok := mapped[entryID]; ok {
				if enc, ok := byURL[url]; ok {
					enc.ID = id
				}
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("storage: returned enclosures: %w", err)
	}
	return nil
}

func (s *Storage) createEnclosure(ctx context.Context, tx pgx.Tx,
	enclosure *model.Enclosure,
) error {
	url := strings.TrimSpace(enclosure.URL)
	if url == "" {
		return nil
	}

	rows, _ := tx.Query(ctx, `
INSERT INTO enclosures
	          (url, size, mime_type, entry_id, user_id, media_progression)
     VALUES ($1,  $2,   $3,        $4,       $5,      $6)
ON CONFLICT (user_id, entry_id, md5(url)) DO NOTHING
  RETURNING id`,
		url,
		enclosure.Size,
		enclosure.MimeType,
		enclosure.EntryID,
		enclosure.UserID,
		enclosure.MediaProgression)

	ret, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToStructByNameLax[model.Enclosure])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf(`unable to create enclosure: %w`, err)
	}
	enclosure.ID = ret.ID
	return nil
}

func (s *Storage) updateEnclosures(ctx context.Context, tx pgx.Tx,
	entry *model.Entry,
) error {
	if len(entry.Enclosures) == 0 {
		return nil
	}

	urls := make([]string, len(entry.Enclosures))
	for i, enclosure := range entry.Enclosures {
		urls[i] = strings.TrimSpace(enclosure.URL)
		if err := s.createEnclosure(ctx, tx, enclosure); err != nil {
			return err
		}
	}

	_, err := tx.Exec(ctx, `
DELETE FROM enclosures WHERE user_id=$1 AND entry_id=$2 AND url <> ALL($3)`,
		entry.UserID, entry.ID, urls)
	if err != nil {
		return fmt.Errorf(`unable to delete old enclosures: %w`, err)
	}
	return nil
}

func (s *Storage) UpdateEnclosure(ctx context.Context,
	enclosure *model.Enclosure,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE enclosures
   SET url=$1,
       size=$2,
       mime_type=$3,
       entry_id=$4,
       user_id=$5,
       media_progression=$6
 WHERE id=$7`,
		enclosure.URL,
		enclosure.Size,
		enclosure.MimeType,
		enclosure.EntryID,
		enclosure.UserID,
		enclosure.MediaProgression,
		enclosure.ID)
	if err != nil {
		return fmt.Errorf(`store: unable to update enclosure #%d : %w`,
			enclosure.ID, err)
	}
	return nil
}
