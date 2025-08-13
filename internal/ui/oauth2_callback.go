// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) oauth2Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logging.FromContext(ctx)

	provider := request.RouteStringParam(r, "provider")
	if provider == "" {
		log.Warn("Invalid or missing OAuth2 provider")
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	code := request.QueryStringParam(r, "code", "")
	if code == "" {
		log.Warn("No code received on OAuth2 callback")
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	sessionData, err := h.sessionData(r)
	if err != nil {
		log.Error("Unable load OAuth2 session data", slog.Any("error", err))
		html.Redirect(w, r, route.Path(h.router, "login"))
		return

	}

	state := request.QueryStringParam(r, "state", "")
	wantState := sessionData.OAuth2State
	stateInvalid := subtle.ConstantTimeCompare([]byte(state),
		[]byte(wantState)) == 0
	if stateInvalid {
		log.Warn("Invalid OAuth2 state value received",
			slog.String("expected", wantState),
			slog.String("received", state))
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	authProvider, err := getOAuth2Manager(ctx).FindProvider(provider)
	if err != nil {
		log.Error("Unable to initialize OAuth2 provider",
			slog.String("provider", provider),
			slog.Any("error", err))
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	profile, err := authProvider.GetProfile(ctx, code,
		sessionData.OAuth2CodeVerifier)
	if err != nil {
		log.Warn("Unable to get OAuth2 profile from provider",
			slog.String("provider", provider),
			slog.Any("error", err))
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	if user := request.User(r); user != nil {
		exists, err := h.store.AnotherUserWithFieldExists(ctx, user.ID, profile.Key,
			profile.ID)
		if err != nil {
			log.Error("unable check another user exists",
				slog.Int64("user_id", user.ID),
				slog.String("field", profile.Key),
				slog.String("value", profile.ID),
				slog.Any("error", err))
			html.ServerError(w, r, err)
			return
		}

		s := session.New(h.store, r).
			SetOAuth2State("").
			SetOAuth2CodeVerifier("")
		printer := locale.NewPrinter(request.UserLanguage(r))

		if exists {
			log.Error(
				"OAuth2 user cannot be associated because it is already associated with another user",
				slog.Int64("user_id", user.ID),
				slog.String("oauth2_provider", provider),
				slog.String("oauth2_profile_id", profile.ID))
			s.NewFlashErrorMessage(printer.Print("error.duplicate_linked_account")).
				Commit(ctx)
			html.Redirect(w, r, route.Path(h.router, "settings"))
			return
		}

		authProvider.PopulateUserWithProfileID(user, profile)
		if err := h.store.UpdateUser(ctx, user); err != nil {
			html.ServerError(w, r, err)
			return
		}

		s.NewFlashMessage(printer.Print("alert.account_linked")).Commit(ctx)
		html.Redirect(w, r, route.Path(h.router, "settings"))
		return
	}

	user, err := h.store.UserByField(ctx, profile.Key, profile.ID)
	if err != nil {
		html.ServerError(w, r, fmt.Errorf(
			"ui: fetch user by OAuth2 profile (%q = %q): %w",
			profile.Key, profile.ID, err))
		return
	}

	if user == nil {
		if !config.Opts.IsOAuth2UserCreationAllowed() {
			html.Forbidden(w, r)
			return
		}

		if h.store.UserExists(ctx, profile.Username) {
			printer := locale.NewPrinter(request.UserLanguage(r))
			html.BadRequest(w, r, errors.New(
				printer.Print("error.user_already_exists")))
			return
		}

		createRequest := &model.UserCreationRequest{
			Username: profile.Username,
		}
		authProvider.PopulateUserCreationWithProfileID(createRequest, profile)

		user, err = h.store.CreateUser(ctx, createRequest)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	clientIP := request.ClientIP(r)
	s, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(), clientIP)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	log.Info("User authenticated successfully using OAuth2",
		slog.Bool("authentication_successful", true),
		slog.String("client_ip", clientIP),
		slog.GroupAttrs("user",
			slog.Int64("id", user.ID),
			slog.String("name", user.Username),
			slog.String("agent", r.UserAgent())),
		slog.String("session_id", s.ID))

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		html.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.ExpiredSessionData())
	http.SetCookie(w, cookie.NewSession(s.ID))
	html.Redirect(w, r, route.Path(h.router, user.DefaultHomePage))
}

func (h *handler) sessionData(r *http.Request) (*model.SessionData, error) {
	if s := request.Session(r); s != nil {
		return s.Data, nil
	}

	plaintext := request.CookieValue(r, cookie.CookieSessionData)
	if plaintext == "" {
		return nil, errors.New("session data cookie not found")
	}

	b, err := h.secureCookie.DecryptCookie(plaintext)
	if err != nil {
		return nil, fmt.Errorf(
			"ui: unable decrypt session data from cookie: %w", err)
	}

	sessionData := new(model.SessionData)
	if err := json.Unmarshal(b, sessionData); err != nil {
		return nil, fmt.Errorf(
			"ui: unable unmarshal session data from cookie: %w", err)
	}
	return sessionData, nil
}
