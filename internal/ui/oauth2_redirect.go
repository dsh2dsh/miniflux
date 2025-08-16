// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/oauth2"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) oauth2Redirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logging.FromContext(ctx)

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
		h.redirect(w, r, "login")
		return
	}

	auth := oauth2.GenerateAuthorization(authProvider.GetConfig())
	if s := request.Session(r); s != nil {
		r = r.WithContext(request.WithSession(ctx, s))
		session.New(h.store, r).
			SetOAuth2State(auth.State()).
			SetOAuth2CodeVerifier(auth.CodeVerifier()).
			Commit(ctx)
		html.Redirect(w, r, auth.RedirectURL())
		return
	}

	sessionData := model.SessionData{
		OAuth2State:        auth.State(),
		OAuth2CodeVerifier: auth.CodeVerifier(),
	}
	if err := h.setSessionDataCookie(w, &sessionData); err != nil {
		log.Error("Unable to set OAuth2 session cookie", slog.Any("error", err))
		h.redirect(w, r, "login")
		return
	}
	html.Redirect(w, r, auth.RedirectURL())
}

func (h *handler) setSessionDataCookie(w http.ResponseWriter,
	data *model.SessionData,
) error {
	b, err := json.Marshal(&data)
	if err != nil {
		return fmt.Errorf("ui: marshal session data to cookie: %w", err)
	}

	encrypted, err := h.secureCookie.EncryptCookie(b)
	if err != nil {
		return fmt.Errorf("ui: encrypt session data cookie: %w", err)
	}

	http.SetCookie(w, cookie.NewSessionData(encrypted))
	return nil
}
