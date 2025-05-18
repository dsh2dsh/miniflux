// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showSessionsPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	sessions, err := h.store.UserSessions(r.Context(), v.User().ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sessions.UseTimezone(v.User().Timezone)

	v.Set("menu", "settings").
		Set("currentSessionToken", request.UserSessionToken(r)).
		Set("sessions", sessions)
	html.OK(w, r, v.Render("sessions"))
}
