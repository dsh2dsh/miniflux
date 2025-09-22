// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showSessionsPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var sessions model.Sessions
	v.Go(func(ctx context.Context) (err error) {
		sessions, err = h.store.UserSessions(ctx, v.UserID())
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	sessions.UseTimezone(v.User().Timezone)
	v.Set("menu", "settings").
		Set("currentSessionToken", request.SessionID(r)).
		Set("sessions", sessions)
	html.OK(w, r, v.Render("sessions"))
}
