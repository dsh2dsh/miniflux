// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/model"
)

// APIKeyExists checks if an API Key with the same description exists.
func (s *Storage) APIKeyExists(ctx context.Context, userID int64,
	description string,
) (bool, error) {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (
  SELECT FROM api_keys
  WHERE user_id=$1 AND lower(description)=lower($2) LIMIT 1)`,
		userID, description)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("storage: failed API key lookup: %w", err)
	}
	return result, nil
}

// SetAPIKeyUsedTimestamp updates the last used date of an API Key.
func (s *Storage) SetAPIKeyUsedTimestamp(ctx context.Context, userID int64,
	token string,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE api_keys SET last_used_at=now() WHERE user_id=$1 and token=$2`,
		userID, token)
	if err != nil {
		return fmt.Errorf(
			`store: unable to update last used date for API key: %w`, err)
	}
	return nil
}

// APIKeys returns all API Keys that belongs to the given user.
func (s *Storage) APIKeys(ctx context.Context, userID int64,
) ([]*model.APIKey, error) {
	rows, _ := s.db.Query(ctx, `
SELECT id, user_id, token, description, last_used_at, created_at
  FROM api_keys
 WHERE user_id=$1
 ORDER BY description ASC`,
		userID)

	apiKeys, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByName[model.APIKey])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch API Keys: %w`, err)
	}
	return apiKeys, nil
}

// CreateAPIKey inserts a new API key.
func (s *Storage) CreateAPIKey(ctx context.Context, apiKey *model.APIKey,
) error {
	rows, _ := s.db.Query(ctx, `
INSERT INTO api_keys (user_id, token, description)
              VALUES ($1,      $2,    $3)
  RETURNING id, created_at`,
		apiKey.UserID, apiKey.Token, apiKey.Description,
	)

	created, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToStructByNameLax[model.APIKey])
	if err != nil {
		return fmt.Errorf(`store: unable to create category: %w`, err)
	}

	apiKey.ID = created.ID
	apiKey.CreatedAt = created.CreatedAt
	return nil
}

// RemoveAPIKey deletes an API Key.
func (s *Storage) RemoveAPIKey(ctx context.Context, userID, keyID int64) error {
	_, err := s.db.Exec(ctx, `
DELETE FROM api_keys WHERE id = $1 AND user_id = $2`,
		keyID, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to remove this API Key: %w`, err)
	}
	return nil
}
