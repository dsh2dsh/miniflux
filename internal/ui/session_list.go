// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showSessionsPage(w http.ResponseWriter, r *http.Request) {
	user, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sessions, err := h.store.UserSessions(r.Context(), user.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sessions.UseTimezone(user.Timezone)

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("currentSessionToken", request.UserSessionToken(r))
	view.Set("sessions", sessions)
	view.Set("menu", "settings")
	view.Set("user", user)
	view.Set("countUnread", h.store.CountUnreadEntries(r.Context(), user.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), user.ID))
	html.OK(w, r, view.Render("sessions"))
}
