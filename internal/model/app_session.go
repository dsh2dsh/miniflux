// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"fmt"
	"time"

	"miniflux.app/v2/internal/timezone"
)

// Session represents a session in the system.
type Session struct {
	ID        string       `db:"id"`
	UserID    int64        `db:"user_id"`
	Data      *SessionData `db:"data"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

func (self *Session) String() string {
	return fmt.Sprintf(`ID=%q, UserID=%v, Data={%v}`, self.ID, self.UserID,
		self.Data)
}

// UseTimezone converts creation date to the given timezone.
func (self *Session) UseTimezone(tz string) {
	timezone.Convert(tz, &self.CreatedAt, &self.UpdatedAt)
}

func (self *Session) Token() string     { return self.ID }
func (self *Session) UserAgent() string { return self.Data.UserAgent }
func (self *Session) IP() string        { return self.Data.IP }

// SessionData represents the data attached to the session.
type SessionData struct {
	OAuth2State         string          `json:"oauth2_state,omitempty"`
	OAuth2CodeVerifier  string          `json:"oauth2_code_verifier,omitempty"`
	FlashMessage        string          `json:"flash_message,omitempty"`
	FlashErrorMessage   string          `json:"flash_error_message,omitempty"`
	Language            string          `json:"language,omitempty"`
	Theme               string          `json:"theme,omitempty"`
	LastForceRefresh    int64           `json:"last_force_refresh,omitempty"`
	WebAuthnSessionData WebAuthnSession `json:"webauthn_session_data,omitzero"`
	UserAgent           string          `json:"user_agent,omitempty"`
	IP                  string          `json:"ip,omitempty"`
}

func (self *SessionData) String() string {
	return fmt.Sprintf(`OAuth2State=%q, OAuth2CodeVerifier=%q, FlashMsg=%q, FlashErrMsg=%q, Lang=%q, Theme=%q, LastForceRefresh=%v, WebAuthnSession=%q`,
		self.OAuth2State,
		self.OAuth2CodeVerifier,
		self.FlashMessage,
		self.FlashErrorMessage,
		self.Language,
		self.Theme,
		self.LastForceRefresh,
		self.WebAuthnSessionData,
	)
}

// Sessions represents a list of sessions.
type Sessions []*Session

// UseTimezone converts creation date of all sessions to the given timezone.
func (self Sessions) UseTimezone(tz string) {
	for _, s := range self {
		s.UseTimezone(tz)
	}
}
