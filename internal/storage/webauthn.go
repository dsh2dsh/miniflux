// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// AddWebAuthnCredential handles storage of webauthn credentials.
func (s *Storage) AddWebAuthnCredential(ctx context.Context, userID int64,
	handle []byte, credential *webauthn.Credential,
) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO webauthn_credentials (
  handle,
  cred_id,
  user_id,
  public_key,
  attestation_type,
  aaguid,
  sign_count,
  clone_warning,
  backup_eligible,
  backup_state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		handle,
		credential.ID,
		userID,
		credential.PublicKey,
		credential.AttestationType,
		credential.Authenticator.AAGUID,
		credential.Authenticator.SignCount,
		credential.Authenticator.CloneWarning,
		credential.Flags.BackupEligible,
		credential.Flags.BackupState,
	)
	if err != nil {
		return fmt.Errorf("storage: unable store wabauthn credential: %w", err)
	}
	return nil
}

func (s *Storage) WebAuthnCredentialByHandle(ctx context.Context, handle []byte,
) (int64, *model.WebAuthnCredential, error) {
	credential := &model.WebAuthnCredential{}
	var userID int64
	var backupEligible pgtype.Bool

	err := s.db.QueryRow(ctx, `
SELECT
	user_id,
	cred_id,
	public_key,
	attestation_type,
	aaguid,
	sign_count,
	clone_warning,
	added_on,
	last_seen_on,
	name,
  backup_eligible,
  backup_state
 FROM webauthn_credentials WHERE handle = $1`, handle).Scan(
		&userID,
		&credential.Credential.ID,
		&credential.Credential.PublicKey,
		&credential.Credential.AttestationType,
		&credential.Credential.Authenticator.AAGUID,
		&credential.Credential.Authenticator.SignCount,
		&credential.Credential.Authenticator.CloneWarning,
		&credential.AddedOn,
		&credential.LastSeenOn,
		&credential.Name,
		&backupEligible,
		&credential.Credential.Flags.BackupState)
	if err != nil {
		return 0, nil, fmt.Errorf(
			"storage: unable fetch webauthn credential: %w", err)
	}

	if backupEligible.Valid {
		credential.Credential.Flags.BackupEligible = backupEligible.Bool
		credential.BackupEligibleKnown = true
	}
	credential.Handle = handle
	return userID, credential, nil
}

func (s *Storage) WebAuthnCredentialsByUserID(ctx context.Context, userID int64,
) ([]model.WebAuthnCredential, error) {
	rows, err := s.db.Query(ctx, `
SELECT
  handle,
  cred_id,
  public_key,
  attestation_type,
  aaguid,
  sign_count,
  clone_warning,
  name,
  added_on,
  last_seen_on,
  backup_eligible,
  backup_state
FROM webauthn_credentials
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"storage: unable fetch webauthn credentials: %w", err)
	}
	defer rows.Close()

	var creds []model.WebAuthnCredential

	for rows.Next() {
		var cred model.WebAuthnCredential
		var backupEligible pgtype.Bool
		err := rows.Scan(
			&cred.Handle,
			&cred.Credential.ID,
			&cred.Credential.PublicKey,
			&cred.Credential.AttestationType,
			&cred.Credential.Authenticator.AAGUID,
			&cred.Credential.Authenticator.SignCount,
			&cred.Credential.Authenticator.CloneWarning,
			&cred.Name,
			&cred.AddedOn,
			&cred.LastSeenOn,
			&backupEligible,
			&cred.Credential.Flags.BackupState)
		if err != nil {
			return nil, fmt.Errorf(
				"storage: unable fetch webauthn credentials: %w", err)
		}

		if backupEligible.Valid {
			cred.Credential.Flags.BackupEligible = backupEligible.Bool
			cred.BackupEligibleKnown = true
		}
		creds = append(creds, cred)
	}
	return creds, nil
}

// WebAuthnSaveLogin writes back the per-assertion fields (sign count, clone
// warning, backup state, BE) the WebAuthn spec requires after every successful
// login.
func (s *Storage) WebAuthnSaveLogin(ctx context.Context, handle []byte,
	credential *webauthn.Credential,
) error {
	_, err := s.db.Exec(ctx, `
UPDATE webauthn_credentials
   SET last_seen_on = NOW(),
       sign_count = $2,
       clone_warning = $3,
       backup_eligible = $4,
       backup_state = $5
 WHERE handle = $1`,
		handle,
		credential.Authenticator.SignCount,
		credential.Authenticator.CloneWarning,
		credential.Flags.BackupEligible,
		credential.Flags.BackupState)
	if err != nil {
		return fmt.Errorf(
			`store: unable to update webauthn credential after login: %w`, err)
	}
	return nil
}

func (s *Storage) WebAuthnUpdateName(ctx context.Context, userID int64,
	handle []byte, name string,
) (int64, error) {
	result, err := s.db.Exec(ctx, `
UPDATE webauthn_credentials
   SET name = $1
 WHERE user_id = $2 AND handle = $3`,
		name, userID, handle)
	if err != nil {
		return 0, fmt.Errorf(
			`storage: update name for webauthn credential: %w`, err)
	}
	return result.RowsAffected(), nil
}

func (s *Storage) CountWebAuthnCredentialsByUserID(ctx context.Context,
	userID int64,
) int {
	rows, _ := s.db.Query(ctx,
		`SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = $1`, userID)

	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if errors.Is(err, pgx.ErrNoRows) {
		return 0
	} else if err != nil {
		logging.FromContext(ctx).Error(
			"store: unable to count webauthn certs for user",
			slog.Int64("user_id", userID),
			slog.Any("error", err))
		return 0
	}
	return count
}

func (s *Storage) DeleteCredentialByHandle(ctx context.Context, userID int64,
	handle []byte,
) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM webauthn_credentials WHERE user_id = $1 AND handle = $2`,
		userID, handle)
	if err != nil {
		return fmt.Errorf("storage: unable delete webauthn credentials: %w", err)
	}
	return nil
}

func (s *Storage) DeleteAllWebAuthnCredentialsByUserID(ctx context.Context,
	userID int64,
) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM webauthn_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("storage: unable delete webauthn credentials: %w", err)
	}
	return nil
}
