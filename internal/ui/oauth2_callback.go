// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"crypto/subtle"
	"errors"
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

	state := request.QueryStringParam(r, "state", "")
	wantState := request.OAuth2State(r)
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
		request.OAuth2CodeVerifier(r))
	if err != nil {
		log.Warn("Unable to get OAuth2 profile from provider",
			slog.String("provider", provider),
			slog.Any("error", err))
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}

	s := request.Session(r)
	if s == nil {
		log.Error("expected session not found")
		html.Redirect(w, r, route.Path(h.router, "login"))
		return
	}
	printer := locale.NewPrinter(request.UserLanguage(r))

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

		if exists {
			log.Error(
				"Oauth2 user cannot be associated because it is already associated with another user",
				slog.Int64("user_id", user.ID),
				slog.String("oauth2_provider", provider),
				slog.String("oauth2_profile_id", profile.ID))
			session.New(h.store, r).
				NewFlashErrorMessage(printer.Print("error.duplicate_linked_account")).
				Commit(ctx)
			html.Redirect(w, r, route.Path(h.router, "settings"))
			return
		}

		authProvider.PopulateUserWithProfileID(user, profile)
		if err := h.store.UpdateUser(ctx, user); err != nil {
			html.ServerError(w, r, err)
			return
		}

		session.New(h.store, r).
			NewFlashMessage(printer.Print("alert.account_linked")).
			Commit(ctx)
		html.Redirect(w, r, route.Path(h.router, "settings"))
		return
	}

	user, err := h.store.UserByField(ctx, profile.Key, profile.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if user == nil {
		if !config.Opts.IsOAuth2UserCreationAllowed() {
			html.Forbidden(w, r)
			return
		}

		if h.store.UserExists(ctx, profile.Username) {
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
	log.Info("User authenticated successfully using OAuth2",
		slog.Bool("authentication_successful", true),
		slog.String("client_ip", clientIP),
		slog.Group("user",
			slog.String("agent", r.UserAgent()),
			slog.Int64("id", user.ID),
			slog.String("name", user.Username)),
		slog.String("session_id", s.ID))

	err = h.store.UpdateAppSessionUserId(ctx, s, user.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		html.ServerError(w, r, err)
		return
	}

	session.New(h.store, r).
		SetLanguage(user.Language).
		SetTheme(user.Theme).
		Commit(ctx)

	http.SetCookie(w, cookie.NewSession(s.ID))
	html.Redirect(w, r, route.Path(h.router, user.DefaultHomePage))
}
