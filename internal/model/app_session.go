// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// SessionData represents the data attached to the session.
type SessionData struct {
	CSRF                string          `json:"csrf"`
	OAuth2State         string          `json:"oauth2_state"`
	OAuth2CodeVerifier  string          `json:"oauth2_code_verifier"`
	FlashMessage        string          `json:"flash_message"`
	FlashErrorMessage   string          `json:"flash_error_message"`
	Language            string          `json:"language"`
	Theme               string          `json:"theme"`
	PocketRequestToken  string          `json:"pocket_request_token"`
	LastForceRefresh    string          `json:"last_force_refresh"`
	WebAuthnSessionData WebAuthnSession `json:"webauthn_session_data"`
}

func (s *SessionData) String() string {
	return fmt.Sprintf(`CSRF=%q, OAuth2State=%q, OAuth2CodeVerifier=%q, FlashMsg=%q, FlashErrMsg=%q, Lang=%q, Theme=%q, PocketTkn=%q, LastForceRefresh=%s, WebAuthnSession=%q`,
		s.CSRF,
		s.OAuth2State,
		s.OAuth2CodeVerifier,
		s.FlashMessage,
		s.FlashErrorMessage,
		s.Language,
		s.Theme,
		s.PocketRequestToken,
		s.LastForceRefresh,
		s.WebAuthnSessionData,
	)
}

// Value converts the session data to JSON.
func (s *SessionData) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("model: failed marshal: %w", err)
	}
	return b, nil
}

// Scan converts raw JSON data.
func (s *SessionData) Scan(src any) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("session: unable to assert type of src")
	}

	if err := json.Unmarshal(source, s); err != nil {
		return fmt.Errorf("session: %w", err)
	}
	return nil
}

// Session represents a session in the system.
type Session struct {
	ID   string
	Data *SessionData
}

func (s *Session) String() string {
	return fmt.Sprintf(`ID=%q, Data={%v}`, s.ID, s.Data)
}
