// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	session.New(h.store, request.SessionID(r)).
		SetLanguage(user.Language).
		SetTheme(user.Theme).
		Commit(r.Context())

	err = h.store.RemoveUserSessionByToken(r.Context(), userID,
		request.UserSessionToken(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.Expired(
		cookie.CookieUserSessionID,
		config.Opts.HTTPS(),
		config.Opts.BasePath(),
	))
	html.Redirect(w, r, route.Path(h.router, "login"))
}
