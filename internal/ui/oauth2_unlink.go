// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) oauth2Unlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logging.FromContext(ctx)

	if config.Opts.DisableLocalAuth() {
		log.Warn("blocking oauth2 unlink attempt, local auth is disabled",
			slog.String("user_agent", r.UserAgent()))
		h.redirect(w, r, "login")
		return
	}

	printer := locale.NewPrinter(request.UserLanguage(r))
	provider := request.RouteStringParam(r, "provider")
	if provider == "" {
		log.Warn("Invalid or missing OAuth2 provider")
		h.redirect(w, r, "login")
		return
	}

	authProvider, err := getOAuth2Manager(ctx).FindProvider(provider)
	if err != nil {
		log.Error("Unable to initialize OAuth2 provider",
			slog.String("provider", provider),
			slog.Any("error", err))
		h.redirect(w, r, "settings")
		return
	}

	user := request.User(r)
	hasPassword, err := h.store.HasPassword(ctx, user.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess := session.New(h.store, r)
	defer sess.Commit(ctx)

	if !hasPassword {
		sess.NewFlashErrorMessage(printer.Print(
			"error.unlink_account_without_password"))
		h.redirect(w, r, "settings")
		return
	}

	authProvider.UnsetUserProfileID(user)
	if err := h.store.UpdateUser(ctx, user); err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess.NewFlashMessage(printer.Print("alert.account_unlinked"))
	h.redirect(w, r, "settings")
}
