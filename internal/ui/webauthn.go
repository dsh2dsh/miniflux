// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
)

type WebAuthnUser struct {
	User        *model.User
	AuthnID     []byte
	Credentials []model.WebAuthnCredential
}

func (u WebAuthnUser) WebAuthnID() []byte {
	return u.AuthnID
}

func (u WebAuthnUser) WebAuthnName() string {
	return u.User.Username
}

func (u WebAuthnUser) WebAuthnDisplayName() string {
	return u.User.Username
}

func (u WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(u.Credentials))
	for i, cred := range u.Credentials {
		creds[i] = cred.Credential
	}
	return creds
}

func newWebAuthn() (*webauthn.WebAuthn, error) {
	url, err := url.Parse(config.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("ui: failed parse %q: %w", config.BaseURL(), err)
	}
	authn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Miniflux",
		RPID:          url.Hostname(),
		RPOrigins:     []string{config.RootURL()},
	})
	if err != nil {
		return nil, fmt.Errorf("ui: failed create webauthn: %w", err)
	}
	return authn, nil
}

func (h *handler) beginRegistration(w http.ResponseWriter, r *http.Request,
) (*protocol.CredentialCreation, error) {
	web, err := newWebAuthn()
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	user := request.User(r)
	if user == nil {
		return nil, response.ErrUnauthorized
	}

	creds, err := h.store.WebAuthnCredentialsByUserID(r.Context(), user.ID)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	credsDescriptors := make([]protocol.CredentialDescriptor, len(creds))
	for i, cred := range creds {
		credsDescriptors[i] = cred.Credential.Descriptor()
	}

	options, sessionData, err := web.BeginRegistration(
		WebAuthnUser{
			user,
			crypto.GenerateRandomBytes(32),
			nil,
		},
		webauthn.WithExclusions(credsDescriptors),
		webauthn.WithResidentKeyRequirement(
			protocol.ResidentKeyRequirementPreferred),
		webauthn.WithExtensions(
			protocol.AuthenticationExtensions{"credProps": true}),
	)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	session.New(h.store, r).
		SetWebAuthnSessionData(&model.WebAuthnSession{SessionData: sessionData}).
		Commit(r.Context())
	return options, nil
}

func (h *handler) finishRegistration(w http.ResponseWriter, r *http.Request,
) (map[string]string, error) {
	web, err := newWebAuthn()
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	user := request.User(r)
	if user == nil {
		return nil, response.ErrUnauthorized
	}

	sessionData := request.WebAuthnSessionData(r)
	webAuthnUser := WebAuthnUser{user, sessionData.UserID, nil}
	cred, err := web.FinishRegistration(webAuthnUser, *sessionData.SessionData, r)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	err = h.store.AddWebAuthnCredential(r.Context(), user.ID, sessionData.UserID,
		cred)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	handleEncoded := model.WebAuthnCredential{Handle: sessionData.UserID}.
		HandleEncoded()
	redirect := route.Path(h.router, "webauthnRename", "credentialHandle",
		handleEncoded)
	return map[string]string{"redirect": redirect}, nil
}

func (h *handler) beginLogin(w http.ResponseWriter, r *http.Request,
) (*protocol.CredentialAssertion, error) {
	web, err := newWebAuthn()
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	ctx := r.Context()
	var user *model.User
	username := request.QueryStringParam(r, "username", "")
	if username != "" {
		user, err = h.store.UserByUsername(ctx, username)
		if err != nil {
			return nil, response.ErrUnauthorized
		}
	}

	var assertion *protocol.CredentialAssertion
	var sessionData *webauthn.SessionData
	if user != nil {
		creds, err := h.store.WebAuthnCredentialsByUserID(ctx, user.ID)
		if err != nil {
			return nil, response.WrapServerError(err)
		}
		assertion, sessionData, err = web.BeginLogin(WebAuthnUser{user, nil, creds})
		if err != nil {
			return nil, response.WrapServerError(err)
		}
	} else {
		assertion, sessionData, err = web.BeginDiscoverableLogin()
		if err != nil {
			return nil, response.WrapServerError(err)
		}
	}

	sessionCookie := model.SessionData{
		WebAuthnSessionData: model.WebAuthnSession{SessionData: sessionData},
	}
	if err := h.setSessionDataCookie(w, &sessionCookie); err != nil {
		return nil, response.WrapServerError(err)
	}
	return assertion, nil
}

func (h *handler) finishLogin(w http.ResponseWriter, r *http.Request) error {
	web, err := newWebAuthn()
	if err != nil {
		return response.WrapServerError(err)
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		return response.WrapServerError(err)
	}

	ctx := r.Context()
	log := logging.FromContext(ctx)
	log.Debug("WebAuthn: parsed response flags",
		slog.Bool("user_present",
			parsedResponse.Response.AuthenticatorData.Flags.HasUserPresent()),
		slog.Bool("user_verified",
			parsedResponse.Response.AuthenticatorData.Flags.HasUserVerified()),
		slog.Bool("has_attested_credential_data",
			parsedResponse.Response.AuthenticatorData.Flags.HasAttestedCredentialData()),
		slog.Bool("has_backup_eligible",
			parsedResponse.Response.AuthenticatorData.Flags.HasBackupEligible()),
		slog.Bool("has_backup_state",
			parsedResponse.Response.AuthenticatorData.Flags.HasBackupState()))

	sessionCookie, err := h.sessionData(r)
	if err != nil {
		return response.WrapServerError(err)
	}
	sessionData := sessionCookie.WebAuthnSessionData

	var user *model.User
	username := request.QueryStringParam(r, "username", "")
	if username != "" {
		user, err = h.store.UserByUsername(ctx, username)
		if err != nil {
			return response.ErrUnauthorized
		}
	}

	var matchingCredential *model.WebAuthnCredential
	if user != nil {
		storedCredentials, err := h.store.WebAuthnCredentialsByUserID(ctx, user.ID)
		if err != nil {
			return response.WrapServerError(err)
		}

		sessionData.UserID = parsedResponse.Response.UserHandle
		webAuthUser := WebAuthnUser{
			user,
			parsedResponse.Response.UserHandle,
			storedCredentials,
		}

		// Since go-webauthn v0.11.0, the backup eligibility flag is strictly
		// validated, but Miniflux does not store this flag. This workaround set the
		// flag based on the parsed response, and avoid "BackupEligible flag
		// inconsistency detected during login validation" error.
		//
		// See https://github.com/go-webauthn/webauthn/pull/240
		for i := range webAuthUser.Credentials {
			webAuthUser.Credentials[i].Credential.Flags.BackupEligible = parsedResponse.Response.AuthenticatorData.Flags.HasBackupEligible()
		}

		for _, webAuthCredential := range webAuthUser.WebAuthnCredentials() {
			log.Debug("WebAuthn: stored credential flags",
				slog.Bool("user_present", webAuthCredential.Flags.UserPresent),
				slog.Bool("user_verified", webAuthCredential.Flags.UserVerified),
				slog.Bool("backup_eligible", webAuthCredential.Flags.BackupEligible),
				slog.Bool("backup_state", webAuthCredential.Flags.BackupState))
		}

		credCredential, err := web.ValidateLogin(webAuthUser,
			*sessionData.SessionData, parsedResponse)
		if err != nil {
			log.Warn("WebAuthn: ValidateLogin failed", slog.Any("error", err))
			return response.ErrUnauthorized
		}

		for _, storedCredential := range storedCredentials {
			if bytes.Equal(credCredential.ID, storedCredential.Credential.ID) {
				matchingCredential = &storedCredential
			}
		}

		if matchingCredential == nil {
			return response.WrapServerError(fmt.Errorf(
				"ui: no matching credential for %v", credCredential))
		}
	} else {
		userByHandle := func(rawID, userHandle []byte) (webauthn.User, error) {
			var uid int64
			uid, matchingCredential, err = h.store.WebAuthnCredentialByHandle(
				ctx, userHandle)
			if err != nil {
				return nil, err
			} else if uid == 0 {
				return nil, fmt.Errorf("ui: no user found for handle %x", userHandle)
			}

			user, err = h.store.UserByID(ctx, uid)
			if err != nil {
				return nil, err
			} else if user == nil {
				return nil, fmt.Errorf("ui: no user found for handle %x", userHandle)
			}

			// Since go-webauthn v0.11.0, the backup eligibility flag is strictly
			// validated, but Miniflux does not store this flag. This workaround set
			// the flag based on the parsed response, and avoid "BackupEligible flag
			// inconsistency detected during login validation" error.
			//
			// See https://github.com/go-webauthn/webauthn/pull/240
			matchingCredential.Credential.Flags.BackupEligible = parsedResponse.Response.AuthenticatorData.Flags.HasBackupEligible()

			return WebAuthnUser{
				user,
				userHandle,
				[]model.WebAuthnCredential{*matchingCredential},
			}, nil
		}

		_, err = web.ValidateDiscoverableLogin(userByHandle,
			*sessionData.SessionData, parsedResponse)
		if err != nil {
			log.Warn("WebAuthn: ValidateDiscoverableLogin failed", slog.Any("error", err))
			return response.ErrUnauthorized
		}
	}

	clientIP := request.ClientIP(r)
	s, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(), clientIP)
	if err != nil {
		return response.WrapServerError(err)
	}

	err = h.store.WebAuthnSaveLogin(ctx, matchingCredential.Handle)
	if err != nil {
		return response.WrapServerError(err)
	}

	log.Info("User authenticated successfully with webauthn",
		slog.String("client_ip", clientIP),
		slog.GroupAttrs("user",
			slog.Int64("id", user.ID),
			slog.String("name", user.Username),
			slog.String("agent", r.UserAgent())),
		slog.String("session_id", s.ID))

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		return response.WrapServerError(err)
	}

	http.SetCookie(w, cookie.ExpiredSessionData())
	http.SetCookie(w, cookie.NewSession(s.ID))
	return nil
}

func (h *handler) renameCredential(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		response.ServerError(w, r, err)
		return
	}

	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}

	cred_uid, cred, err := h.store.WebAuthnCredentialByHandle(
		r.Context(), credentialHandle)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}

	if cred_uid != v.User().ID {
		response.Forbidden(w, r)
		return
	}

	webauthnForm := form.WebauthnForm{Name: cred.Name}

	v.Set("menu", "settings").
		Set("form", webauthnForm).
		Set("cred", cred)
	response.HTML(w, r, v.Render("webauthn_rename"))
}

func (h *handler) saveCredential(w http.ResponseWriter, r *http.Request) {
	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}

	newName := r.FormValue("name")
	err = h.store.WebAuthnUpdateName(r.Context(), credentialHandle, newName)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "settings")
}

func (h *handler) deleteCredential(w http.ResponseWriter, r *http.Request,
) error {
	uid := request.UserID(r)
	if uid == 0 {
		return response.ErrUnauthorized
	}

	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		return response.WrapServerError(err)
	}

	err = h.store.DeleteCredentialByHandle(r.Context(), uid, credentialHandle)
	if err != nil {
		return response.WrapServerError(err)
	}
	return nil
}

func (h *handler) deleteAllCredentials(w http.ResponseWriter, r *http.Request,
) error {
	err := h.store.DeleteAllWebAuthnCredentialsByUserID(r.Context(),
		request.UserID(r))
	if err != nil {
		return response.WrapServerError(err)
	}
	return nil
}
