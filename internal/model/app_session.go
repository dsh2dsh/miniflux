// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import "fmt"

// SessionData represents the data attached to the session.
type SessionData struct {
	CSRF                string          `json:"csrf,omitempty"`
	OAuth2State         string          `json:"oauth2_state,omitempty"`
	OAuth2CodeVerifier  string          `json:"oauth2_code_verifier,omitempty"`
	FlashMessage        string          `json:"flash_message,omitempty"`
	FlashErrorMessage   string          `json:"flash_error_message,omitempty"`
	Language            string          `json:"language,omitempty"`
	Theme               string          `json:"theme,omitempty"`
	PocketRequestToken  string          `json:"pocket_request_token,omitempty"`
	LastForceRefresh    int64           `json:"last_force_refresh,omitempty"`
	WebAuthnSessionData WebAuthnSession `json:"webauthn_session_data,omitzero"`
}

func (s *SessionData) String() string {
	return fmt.Sprintf(`CSRF=%q, OAuth2State=%q, OAuth2CodeVerifier=%q, FlashMsg=%q, FlashErrMsg=%q, Lang=%q, Theme=%q, PocketTkn=%q, LastForceRefresh=%v, WebAuthnSession=%q`,
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

// Session represents a session in the system.
type Session struct {
	ID   string       `db:"id"`
	Data *SessionData `db:"data"`
}

func (s *Session) String() string {
	return fmt.Sprintf(`ID=%q, Data={%v}`, s.ID, s.Data)
}
