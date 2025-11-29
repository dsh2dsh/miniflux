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
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/response/json"
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

func (h *handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
	web, err := newWebAuthn()
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	user := request.User(r)
	if user == nil {
		json.Unauthorized(w, r)
		return
	}

	creds, err := h.store.WebAuthnCredentialsByUserID(r.Context(), user.ID)
	if err != nil {
		json.ServerError(w, r, err)
		return
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
		json.ServerError(w, r, err)
		return
	}

	session.New(h.store, r).
		SetWebAuthnSessionData(&model.WebAuthnSession{SessionData: sessionData}).
		Commit(r.Context())
	json.OK(w, r, options)
}

func (h *handler) finishRegistration(w http.ResponseWriter, r *http.Request) {
	web, err := newWebAuthn()
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	user := request.User(r)
	if user == nil {
		json.Unauthorized(w, r)
		return
	}

	sessionData := request.WebAuthnSessionData(r)
	webAuthnUser := WebAuthnUser{user, sessionData.UserID, nil}
	cred, err := web.FinishRegistration(webAuthnUser, *sessionData.SessionData, r)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	err = h.store.AddWebAuthnCredential(r.Context(), user.ID, sessionData.UserID,
		cred)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	handleEncoded := model.WebAuthnCredential{Handle: sessionData.UserID}.
		HandleEncoded()
	redirect := route.Path(h.router, "webauthnRename", "credentialHandle",
		handleEncoded)
	json.OK(w, r, map[string]string{"redirect": redirect})
}

func (h *handler) beginLogin(w http.ResponseWriter, r *http.Request) {
	web, err := newWebAuthn()
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	ctx := r.Context()
	var user *model.User
	username := request.QueryStringParam(r, "username", "")
	if username != "" {
		user, err = h.store.UserByUsername(ctx, username)
		if err != nil {
			json.Unauthorized(w, r)
			return
		}
	}

	var assertion *protocol.CredentialAssertion
	var sessionData *webauthn.SessionData
	if user != nil {
		creds, err := h.store.WebAuthnCredentialsByUserID(ctx, user.ID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
		assertion, sessionData, err = web.BeginLogin(WebAuthnUser{user, nil, creds})
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	} else {
		assertion, sessionData, err = web.BeginDiscoverableLogin()
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
	}

	sessionCookie := model.SessionData{
		WebAuthnSessionData: model.WebAuthnSession{SessionData: sessionData},
	}
	if err := h.setSessionDataCookie(w, &sessionCookie); err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, assertion)
}

func (h *handler) finishLogin(w http.ResponseWriter, r *http.Request) {
	web, err := newWebAuthn()
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		json.ServerError(w, r, err)
		return
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
		json.ServerError(w, r, err)
		return
	}
	sessionData := sessionCookie.WebAuthnSessionData

	var user *model.User
	username := request.QueryStringParam(r, "username", "")
	if username != "" {
		user, err = h.store.UserByUsername(ctx, username)
		if err != nil {
			json.Unauthorized(w, r)
			return
		}
	}

	var matchingCredential *model.WebAuthnCredential
	if user != nil {
		storedCredentials, err := h.store.WebAuthnCredentialsByUserID(ctx, user.ID)
		if err != nil {
			json.ServerError(w, r, err)
			return
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
			json.Unauthorized(w, r)
			return
		}

		for _, storedCredential := range storedCredentials {
			if bytes.Equal(credCredential.ID, storedCredential.Credential.ID) {
				matchingCredential = &storedCredential
			}
		}

		if matchingCredential == nil {
			json.ServerError(w, r,
				fmt.Errorf("ui: no matching credential for %v", credCredential))
			return
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
			json.Unauthorized(w, r)
			return
		}
	}

	clientIP := request.ClientIP(r)
	s, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(), clientIP)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	err = h.store.WebAuthnSaveLogin(ctx, matchingCredential.Handle)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	log.Info("User authenticated successfully with webauthn",
		slog.String("client_ip", clientIP),
		slog.GroupAttrs("user",
			slog.Int64("id", user.ID),
			slog.String("name", user.Username),
			slog.String("agent", r.UserAgent())),
		slog.String("session_id", s.ID))

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		json.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.ExpiredSessionData())
	http.SetCookie(w, cookie.NewSession(s.ID))
	json.NoContent(w, r)
}

func (h *handler) renameCredential(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	cred_uid, cred, err := h.store.WebAuthnCredentialByHandle(
		r.Context(), credentialHandle)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if cred_uid != v.User().ID {
		html.Forbidden(w, r)
		return
	}

	webauthnForm := form.WebauthnForm{Name: cred.Name}

	v.Set("menu", "settings").
		Set("form", webauthnForm).
		Set("cred", cred)
	html.OK(w, r, v.Render("webauthn_rename"))
}

func (h *handler) saveCredential(w http.ResponseWriter, r *http.Request) {
	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	newName := r.FormValue("name")
	err = h.store.WebAuthnUpdateName(r.Context(), credentialHandle, newName)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "settings")
}

func (h *handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	uid := request.UserID(r)
	if uid == 0 {
		json.Unauthorized(w, r)
		return
	}

	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	err = h.store.DeleteCredentialByHandle(r.Context(), uid, credentialHandle)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	json.NoContent(w, r)
}

func (h *handler) deleteAllCredentials(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteAllWebAuthnCredentialsByUserID(r.Context(),
		request.UserID(r))
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}
