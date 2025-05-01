// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// UserSessions returns the list of sessions for the given user.
func (s *Storage) UserSessions(ctx context.Context, userID int64,
) (model.UserSessions, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, token, created_at, user_agent, abbrev(ip) as ip
  FROM user_sessions
 WHERE user_id=$1 ORDER BY id DESC`,
		userID)

	sessions, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.UserSession])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch user sessions: %w`, err)
	}
	return sessions, nil
}

// CreateUserSessionFromUsername creates a new user session.
func (s *Storage) CreateUserSessionFromUsername(ctx context.Context, username,
	userAgent, ip string,
) (sessionID string, userID int64, _ error) {
	sessionID = crypto.GenerateRandomString(64)

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, _ := tx.Query(ctx,
			`SELECT id FROM users WHERE username = LOWER($1)`, username)
		id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64])
		if err != nil {
			return fmt.Errorf(`fetch user ID: %w`, err)
		}
		userID = id

		_, err = tx.Exec(ctx, `
INSERT INTO user_sessions (token, user_id, user_agent, ip)
                   VALUES ($1,    $2,      $3,         $4)`,
			sessionID, userID, userAgent, ip)
		if err != nil {
			return fmt.Errorf(`store user session: %w`, err)
		}
		return nil
	})
	if err != nil {
		return "", 0, fmt.Errorf(`storage: unable to create user session: %w`, err)
	}
	return sessionID, userID, nil
}

// UserSessionByToken finds a session by the token.
func (s *Storage) UserSessionByToken(ctx context.Context, token string,
) (*model.UserSession, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, token, created_at, user_agent, abbrev(ip) as ip
  FROM user_sessions
 WHERE token = $1`,
		token)

	session, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.UserSession])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch user session: %w`, err)
	}
	return session, nil
}

// RemoveUserSessionByToken remove a session by using the token.
func (s *Storage) RemoveUserSessionByToken(ctx context.Context, userID int64,
	token string,
) error {
	result, err := s.db.Exec(ctx,
		`DELETE FROM user_sessions WHERE user_id=$1 AND token=$2`,
		userID, token)
	if err != nil {
		return fmt.Errorf(`store: unable to remove this user session: %w`, err)
	}

	if result.RowsAffected() != 1 {
		return errors.New(`store: nothing has been removed`)
	}
	return nil
}

// RemoveUserSessionByID remove a session by using the ID.
func (s *Storage) RemoveUserSessionByID(ctx context.Context, userID,
	sessionID int64,
) error {
	result, err := s.db.Exec(ctx,
		`DELETE FROM user_sessions WHERE user_id=$1 AND id=$2`,
		userID, sessionID)
	if err != nil {
		return fmt.Errorf(`store: unable to remove this user session: %w`, err)
	}

	if result.RowsAffected() != 1 {
		return errors.New(`store: nothing has been removed`)
	}
	return nil
}

// CleanOldUserSessions removes user sessions older than specified days.
func (s *Storage) CleanOldUserSessions(ctx context.Context, days int) int64 {
	result, err := s.db.Exec(ctx,
		`DELETE FROM user_sessions WHERE created_at < now() - $1::interval`,
		strconv.FormatInt(int64(days), 10)+" days")
	if err != nil {
		logging.FromContext(ctx).Error("storage: unable clean old user sessions",
			slog.Any("error", err))
		return 0
	}
	return result.RowsAffected()
}
