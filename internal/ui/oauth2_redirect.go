// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/oauth2"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) oauth2Redirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logging.FromContext(ctx)

	provider := request.RouteStringParam(r, "provider")
	if provider == "" {
		log.Warn("Invalid or missing OAuth2 provider")
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

	s := request.Session(r)
	if s == nil {
		s, err = h.store.CreateAppSession(ctx, r.UserAgent(), request.ClientIP(r))
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	auth := oauth2.GenerateAuthorization(authProvider.GetConfig())
	r = r.WithContext(request.WithSession(ctx, s))
	session.New(h.store, r).
		SetOAuth2State(auth.State()).
		SetOAuth2CodeVerifier(auth.CodeVerifier()).
		Commit(ctx)
	html.Redirect(w, r, auth.RedirectURL())
}
