// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
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
	baseURL, err := url.Parse(config.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("ui: failed parse %q: %w", config.BaseURL(), err)
	}
	authn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Miniflux",
		RPID:          baseURL.Hostname(),
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

	credentials, err := h.store.WebAuthnCredentialsByUserID(r.Context(), user.ID)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	creadentialDescriptors := make([]protocol.CredentialDescriptor, len(credentials))
	for i, credential := range credentials {
		creadentialDescriptors[i] = credential.Credential.Descriptor()
	}

	options, sessionData, err := web.BeginRegistration(
		WebAuthnUser{
			User:    user,
			AuthnID: crypto.GenerateRandomBytes(32),
		},
		webauthn.WithExclusions(creadentialDescriptors),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	session.FromContext(r.Context()).
		SetWebAuthnSessionData(&model.WebAuthnSession{SessionData: sessionData})
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
	webAuthnUser := WebAuthnUser{User: user, AuthnID: sessionData.UserID}
	credential, err := web.FinishRegistration(webAuthnUser, *sessionData.SessionData, r)
	if err != nil {
		return nil, response.WrapServerError(err)
	}

	err = h.store.AddWebAuthnCredential(r.Context(), user.ID, sessionData.UserID,
		credential)
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

	assertion, sessionData, err := web.BeginDiscoverableLogin()
	if err != nil {
		return nil, response.WrapServerError(err)
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
	sessionData := sessionCookie.WebAuthnSessionData.SessionData

	var resolvedUser *model.User
	var resolvedCredential *model.WebAuthnCredential

	userByHandle := func(rawID, userHandle []byte) (webauthn.User, error) {
		userID, credential, err := h.store.WebAuthnCredentialByHandle(ctx,
			userHandle)
		if err != nil {
			return nil, err
		} else if userID == 0 || credential == nil {
			return nil, fmt.Errorf("ui: no user found for handle %x", userHandle)
		}

		loadedUser, err := h.store.UserByID(ctx, userID)
		if err != nil {
			return nil, err
		} else if loadedUser == nil {
			return nil, fmt.Errorf("ui: no user found for handle %x", userHandle)
		}

		// One-shot backfill for credentials registered before the
		// backup_eligible column was added: trust the assertion's BE
		// once, then persist it after successful validation.
		if !credential.BackupEligibleKnown {
			credential.Credential.Flags.BackupEligible = parsedResponse.Response.AuthenticatorData.Flags.HasBackupEligible()
		}

		resolvedUser = loadedUser
		resolvedCredential = credential
		return WebAuthnUser{
			User:        loadedUser,
			AuthnID:     userHandle,
			Credentials: []model.WebAuthnCredential{*credential},
		}, nil
	}

	validatedCredential, err := web.ValidateDiscoverableLogin(userByHandle,
		*sessionData, parsedResponse)
	if err != nil {
		log.Warn("WebAuthn: ValidateDiscoverableLogin failed",
			slog.String("client_ip", request.ClientIP(r)),
			slog.String("user_agent", r.UserAgent()),
			slog.Any("error", err))
		return response.ErrUnauthorized
	}

	user := resolvedUser
	matchingCredential := resolvedCredential

	clientIP := request.ClientIP(r)
	s, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(), clientIP)
	if err != nil {
		return response.WrapServerError(err)
	}

	err = h.store.WebAuthnSaveLogin(ctx, matchingCredential.Handle,
		validatedCredential)
	if err != nil {
		slog.Warn("WebAuthn: unable to update last seen date for credential",
			slog.Int64("user_id", user.ID),
			slog.Any("error", err))
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
		slog.Warn("Unable to update last login date",
			slog.Int64("user_id", user.ID),
			slog.Any("error", err))
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

	credUserID, credential, err := h.store.WebAuthnCredentialByHandle(
		r.Context(), credentialHandle)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}

	if credUserID != v.User().ID {
		response.Forbidden(w, r)
		return
	}

	v.Set("menu", "settings").
		Set("form", form.WebauthnForm{Name: credential.Name}).
		Set("cred", credential)
	response.HTML(w, r, v.Render("webauthn_rename"))
}

func (h *handler) saveCredential(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()
	credUserID, credential, err := h.store.WebAuthnCredentialByHandle(ctx,
		credentialHandle)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if credUserID != v.User().ID {
		response.Forbidden(w, r)
		return
	}

	webauthnForm := form.NewWebauthnForm(r)
	if lerr := webauthnForm.Validate(); lerr != nil {
		v.Set("form", webauthnForm)
		v.Set("cred", credential)
		v.Set("menu", "settings")
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		response.HTML(w, r, v.Render("webauthn_rename"))
		return
	}

	rowsAffected, err := h.store.WebAuthnUpdateName(ctx, request.UserID(r),
		credentialHandle, webauthnForm.Name)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if rowsAffected == 0 {
		response.NotFound(w, r)
		return
	}
	h.redirect(w, r, "settings")
}

func (h *handler) deleteCredential(w http.ResponseWriter, r *http.Request,
) error {
	credentialHandleEncoded := request.RouteStringParam(r, "credentialHandle")
	credentialHandle, err := hex.DecodeString(credentialHandleEncoded)
	if err != nil {
		return response.WrapServerError(err)
	}

	err = h.store.DeleteCredentialByHandle(r.Context(), request.UserID(r),
		credentialHandle)
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
