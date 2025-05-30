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

// CreateAppSessionWithUserPrefs creates a new application session with the
// given user preferences.
func (s *Storage) CreateAppSessionWithUserPrefs(ctx context.Context,
	userID int64,
) (*model.Session, error) {
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.CreateAppSessionForUser(ctx, user, "", "")
}

func (s *Storage) CreateAppSessionForUser(ctx context.Context, user *model.User,
	userAgent, ip string,
) (*model.Session, error) {
	session := &model.Session{
		ID:     crypto.GenerateRandomString(32),
		UserID: user.ID,
		Data: &model.SessionData{
			CSRF:      crypto.GenerateRandomString(64),
			Theme:     user.Theme,
			Language:  user.Language,
			UserAgent: userAgent,
			IP:        ip,
		},
	}
	return s.createAppSession(ctx, session)
}

// CreateAppSession creates a new application session.
func (s *Storage) CreateAppSession(ctx context.Context, userAgent, ip string,
) (*model.Session, error) {
	session := &model.Session{
		ID: crypto.GenerateRandomString(32),
		Data: &model.SessionData{
			CSRF:      crypto.GenerateRandomString(64),
			UserAgent: userAgent,
			IP:        ip,
		},
	}
	return s.createAppSession(ctx, session)
}

func (s *Storage) createAppSession(ctx context.Context, sess *model.Session,
) (*model.Session, error) {
	_, err := s.db.Exec(ctx,
		`INSERT INTO sessions (id, user_id, data) VALUES ($1, $2, $3)`,
		sess.ID, sess.UserID, sess.Data)
	if err != nil {
		return nil, fmt.Errorf("storage: create app session: %w", err)
	}
	return sess, nil
}

func (s *Storage) UpdateAppSession(ctx context.Context, sessionID string,
	values map[string]any,
) error {
	_, err := s.db.Exec(ctx,
		`UPDATE sessions SET data = data || $1::jsonb WHERE id=$2`,
		values, sessionID)
	if err != nil {
		return fmt.Errorf(`store: unable to update session field: %w`, err)
	}
	return nil
}

// AppSession returns the given session.
func (s *Storage) AppSession(ctx context.Context, id string,
) (*model.Session, error) {
	rows, _ := s.db.Query(ctx,
		`SELECT id, user_id, data, created_at FROM sessions WHERE id=$1`, id)

	sess, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Session])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("storage: fetch session: %w", err)
	}
	return sess, nil
}

// FlushAllSessions removes all sessions from the database.
func (s *Storage) FlushAllSessions(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions`)
	if err != nil {
		return fmt.Errorf("storage: failed flush sessions: %w", err)
	}
	return nil
}

// CleanOldSessions removes sessions older than specified days.
func (s *Storage) CleanOldSessions(ctx context.Context, days int) int64 {
	result, err := s.db.Exec(ctx,
		`DELETE FROM sessions WHERE created_at < now() - $1::interval`,
		strconv.FormatInt(int64(days), 10)+" days")
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

func (s *Storage) UpdateAppSessionUserId(ctx context.Context, sessionID string,
	userID int64,
) error {
	_, err := s.db.Exec(ctx,
		`UPDATE sessions SET user_id = $1 WHERE id=$2`, userID, sessionID)
	if err != nil {
		return fmt.Errorf("storage: update session user_id field: %w", err)
	}
	return nil
}
