// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package session // import "miniflux.app/v2/internal/ui/session"

import (
	"context"
	"time"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/storage"
)

// Session handles session data.
type Session struct {
	store     *storage.Storage
	sessionID string
}

// New returns a new session handler.
func New(store *storage.Storage, sessionID string) *Session {
	return &Session{store, sessionID}
}

func (s *Session) SetLastForceRefresh(ctx context.Context) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
		"last_force_refresh", time.Now().UTC().Unix())
}

func (s *Session) SetOAuth2State(ctx context.Context, state string) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID, "oauth2_state", state)
}

func (s *Session) SetOAuth2CodeVerifier(ctx context.Context,
	codeVerfier string,
) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
		"oauth2_code_verifier", codeVerfier)
}

// NewFlashMessage creates a new flash message.
func (s *Session) NewFlashMessage(ctx context.Context, message string) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
		"flash_message", message)
}

// FlashMessage returns the current flash message if any.
func (s *Session) FlashMessage(ctx context.Context, message string) string {
	if message != "" {
		_ = s.store.UpdateAppSessionField(ctx, s.sessionID, "flash_message", "")
	}
	return message
}

// NewFlashErrorMessage creates a new flash error message.
func (s *Session) NewFlashErrorMessage(ctx context.Context, message string) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
		"flash_error_message", message)
}

// FlashErrorMessage returns the last flash error message if any.
func (s *Session) FlashErrorMessage(ctx context.Context, message string,
) string {
	if message != "" {
		_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
			"flash_error_message", "")
	}
	return message
}

// SetLanguage updates the language field in session.
func (s *Session) SetLanguage(ctx context.Context, language string) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID, "language", language)
}

// SetTheme updates the theme field in session.
func (s *Session) SetTheme(ctx context.Context, theme string) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID, "theme", theme)
}

// SetPocketRequestToken updates Pocket Request Token.
func (s *Session) SetPocketRequestToken(ctx context.Context,
	requestToken string,
) {
	_ = s.store.UpdateAppSessionField(ctx, s.sessionID,
		"pocket_request_token", requestToken)
}

func (s *Session) SetWebAuthnSessionData(ctx context.Context,
	sessionData *model.WebAuthnSession,
) {
	_ = s.store.UpdateAppSessionObjectField(ctx, s.sessionID,
		"webauthn_session_data", sessionData)
}
