// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnSession handles marshalling / unmarshalling session data
type WebAuthnSession struct {
	*webauthn.SessionData
}

func (s WebAuthnSession) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("model: failed marshal: %w", err)
	}
	return b, nil
}

func (s *WebAuthnSession) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("model: failed unmarshal: %w", err)
	}
	return nil
}

func (s WebAuthnSession) String() string {
	if s.SessionData == nil {
		return "{}"
	}
	return fmt.Sprintf("{Challenge: %s, UserID: %x}", s.Challenge, s.UserID)
}

type WebAuthnCredential struct {
	Credential webauthn.Credential
	Name       string
	AddedOn    *time.Time
	LastSeenOn *time.Time
	Handle     []byte
}

func (s WebAuthnCredential) HandleEncoded() string {
	return hex.EncodeToString(s.Handle)
}
