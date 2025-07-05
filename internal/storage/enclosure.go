// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"

	"miniflux.app/v2/internal/model"
)

func (s *Storage) UpdateEnclosureAt(ctx context.Context, userID, entryID int64,
	enclosure *model.Enclosure, at int,
) (bool, error) {
	result, err := s.db.Exec(ctx, `
UPDATE entries
   SET extra['enclosures'][$1::int]['media_progression'] = $2
WHERE id = $3 AND user_id = $4 AND
      jsonb_array_length(extra['enclosures']) > $1`,
		at, enclosure.MediaProgression,
		entryID, userID)
	if err != nil {
		return false, fmt.Errorf("storage: update enclosure at %d: %w", at, err)
	}
	return result.RowsAffected() != 0, nil
}
