// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

func (s *Storage) CreateAppSessionForUser(ctx context.Context, user *model.User,
	userAgent, ip string,
) (*model.Session, error) {
	session := &model.Session{
		ID:     rand.Text(),
		UserID: user.ID,
		Data: &model.SessionData{
			Theme:     user.Theme,
			Language:  user.Language,
			UserAgent: userAgent,
			IP:        ip,
		},
	}
	return s.createAppSession(ctx, session)
}

func (s *Storage) createAppSession(ctx context.Context, sess *model.Session,
) (*model.Session, error) {
	err := s.db.QueryRow(ctx, `
INSERT INTO sessions (id, user_id, data)
              VALUES ($1, $2,      $3)
RETURNING created_at, updated_at`,
		sess.ID, sess.UserID, &sess.Data).Scan(&sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: create app session: %w", err)
	}
	return sess, nil
}

func (s *Storage) UpdateAppSession(ctx context.Context, m *model.Session,
) error {
	err := s.db.QueryRow(ctx, `
UPDATE sessions
   SET updated_at = now(), data = $2
 WHERE id = $1
RETURNING updated_at`, m.ID, m.Data).Scan(&m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("storage: unable to update session data: %w", err)
	}
	return nil
}

func (s *Storage) RefreshAppSession(ctx context.Context, m *model.Session,
) error {
	err := s.db.QueryRow(ctx, `
UPDATE sessions
   SET updated_at = now()
 WHERE id = $1 RETURNING updated_at`, m.ID).Scan(&m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("storage: unable to refresh session time: %w", err)
	}
	return nil
}

// FlushAllSessions removes all sessions from the database.
func (s *Storage) FlushAllSessions(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions`)
	if err != nil {
		return fmt.Errorf("storage: failed flush sessions: %w", err)
	}
	return nil
}

// CleanOldSessions removes sessions older than specified days or inactiveDays.
func (s *Storage) CleanOldSessions(ctx context.Context, days, inactiveDays int,
) int64 {
	var sql string
	var args []any

	if days > 0 {
		sql = `
DELETE FROM sessions
 WHERE created_at < now() - $1::interval OR
       updated_at < now() - $2::interval`
		args = []any{
			strconv.FormatInt(int64(days), 10) + " days",
			strconv.FormatInt(int64(inactiveDays), 10) + " days",
		}
	} else {
		sql = "DELETE FROM sessions WHERE updated_at < now() - $1::interval"
		args = []any{strconv.FormatInt(int64(inactiveDays), 10) + " days"}
	}

	result, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		logging.FromContext(ctx).Error(
			"storage: unable clean old sessions", slog.Any("error", err))
		return 0
	}
	return result.RowsAffected()
}

func (s *Storage) RemoveAppSessionByID(ctx context.Context, id string) error {
	result, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("storage: unable to remove this app session: %w", err)
	}

	if result.RowsAffected() != 1 {
		return errors.New("storage: no app sessions has been removed")
	}
	return nil
}

// UserSessions returns the list of sessions for the given user.
func (s *Storage) UserSessions(ctx context.Context, userID int64,
) (model.Sessions, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, data, created_at, updated_at
  FROM sessions
 WHERE user_id = $1 ORDER BY updated_at DESC, created_at DESC`, userID)
	sessions, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.Session])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch user sessions: %w`, err)
	}
	return sessions, nil
}
