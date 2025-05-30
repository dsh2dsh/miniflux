// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/model"
)

// UserSessions returns the list of sessions for the given user.
func (s *Storage) UserSessions(ctx context.Context, userID int64,
) (model.UserSessions, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, data, created_at
  FROM sessions
 WHERE user_id = $1 ORDER BY created_at DESC`, userID)

	sessions, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.Session])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch user sessions: %w`, err)
	}

	userSessions := make(model.UserSessions, len(sessions))
	for i, s := range sessions {
		userSessions[i] = &model.UserSession{
			UserID:    s.UserID,
			Token:     s.ID,
			CreatedAt: s.CreatedAt,
			UserAgent: s.Data.UserAgent,
			IP:        s.Data.IP,
		}
	}
	return userSessions, nil
}
