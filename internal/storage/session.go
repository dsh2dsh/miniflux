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

	session := &model.Session{
		ID: crypto.GenerateRandomString(32),
		Data: &model.SessionData{
			CSRF:     crypto.GenerateRandomString(64),
			Theme:    user.Theme,
			Language: user.Language,
		},
	}
	return s.createAppSession(ctx, session)
}

// CreateAppSession creates a new application session.
func (s *Storage) CreateAppSession(ctx context.Context,
) (*model.Session, error) {
	session := &model.Session{
		ID: crypto.GenerateRandomString(32),
		Data: &model.SessionData{
			CSRF: crypto.GenerateRandomString(64),
		},
	}
	return s.createAppSession(ctx, session)
}

func (s *Storage) createAppSession(ctx context.Context, session *model.Session,
) (*model.Session, error) {
	_, err := s.db.Exec(ctx, `INSERT INTO sessions (id, data) VALUES ($1, $2)`,
		session.ID, session.Data)
	if err != nil {
		return nil, fmt.Errorf(`store: unable to create app session: %w`, err)
	}
	return session, nil
}

// UpdateAppSessionField updates only one session field.
func (s *Storage) UpdateAppSessionField(ctx context.Context, sessionID,
	field string, value any,
) error {
	query := `UPDATE sessions SET data['%s'] = to_jsonb($1::text) WHERE id=$2`
	_, err := s.db.Exec(ctx, fmt.Sprintf(query, field), value, sessionID)
	if err != nil {
		return fmt.Errorf(`store: unable to update session field: %w`, err)
	}
	return nil
}

func (s *Storage) UpdateAppSessionObjectField(ctx context.Context, sessionID,
	field string, value any,
) error {
	query := `UPDATE sessions SET data['%s'] = $1 WHERE id=$2`
	_, err := s.db.Exec(ctx, fmt.Sprintf(query, field), value, sessionID)
	if err != nil {
		return fmt.Errorf(`store: unable to update session field: %w`, err)
	}
	return nil
}

// AppSession returns the given session.
func (s *Storage) AppSession(ctx context.Context, id string,
) (*model.Session, error) {
	rows, _ := s.db.Query(ctx, `SELECT id, data FROM sessions WHERE id=$1`, id)

	session, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[model.Session])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf(`store: session not found: %s`, id)
	} else if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch session: %w`, err)
	}
	return session, nil
}

// FlushAllSessions removes all sessions from the database.
func (s *Storage) FlushAllSessions(ctx context.Context) error {
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM user_sessions`)
		if err != nil {
			return fmt.Errorf("full user_sessions: %w", err)
		}

		_, err = tx.Exec(ctx, `DELETE FROM sessions`)
		if err != nil {
			return fmt.Errorf("flush sessions: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("storage: failed flush all sessions: %w", err)
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
