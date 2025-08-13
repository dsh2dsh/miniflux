// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) checkLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientIP := request.ClientIP(r)
	log := logging.FromContext(ctx).With(
		slog.String("client_ip", clientIP),
		slog.String("user_agent", r.UserAgent()))
	v := view.New(h.tpl, r, nil)

	if config.Opts.DisableLocalAuth() {
		log.Warn("blocking local auth login attempt, local auth is disabled")
		html.OK(w, r, v.Render("login"))
		return
	}

	f := form.NewAuthForm(r)
	v.Set("errorMessage",
		locale.NewLocalizedError("error.bad_credentials").
			Translate(request.UserLanguage(r)))
	v.Set("form", f)
	log = log.With(slog.String("username", f.Username))

	if lerr := f.Validate(); lerr != nil {
		translatedErrorMessage := lerr.Translate(request.UserLanguage(r))
		log.Warn("Validation error during login check",
			slog.Any("error", translatedErrorMessage))
		html.OK(w, r, v.Render("login"))
		return
	}

	err := h.store.CheckPassword(ctx, f.Username, f.Password)
	if err != nil {
		log.Warn("Incorrect username or password", slog.Any("error", err))
		html.OK(w, r, v.Render("login"))
		return
	}

	user, err := h.store.UserByUsername(ctx, f.Username)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if user == nil {
		log.Warn("User not found")
		html.OK(w, r, v.Render("login"))
		return
	}
	log.Info("User authenticated successfully with username/password",
		slog.Int64("user_id", user.ID))

	sess, err := h.store.CreateAppSessionForUser(ctx, user, r.UserAgent(),
		clientIP)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if err := h.store.SetLastLogin(ctx, user.ID); err != nil {
		html.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.NewSession(sess.ID))
	html.Redirect(w, r, route.Path(h.router, user.DefaultHomePage))
}
