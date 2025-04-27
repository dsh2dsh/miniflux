// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"time"

	"miniflux.app/v2/internal/crypto"
)

// APIKey represents an application API key.
type APIKey struct {
	ID          int64      `db:"id"`
	UserID      int64      `db:"user_id"`
	Token       string     `db:"token"`
	Description string     `db:"description"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	CreatedAt   time.Time  `db:"created_at"`
}

// NewAPIKey initializes a new APIKey.
func NewAPIKey(userID int64, description string) *APIKey {
	return &APIKey{
		UserID:      userID,
		Token:       crypto.GenerateRandomString(32),
		Description: description,
	}
}
