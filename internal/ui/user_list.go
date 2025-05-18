// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showUsersPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	if !v.User().IsAdmin {
		html.Forbidden(w, r)
		return
	}

	users, err := h.store.Users(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	users.UseTimezone(v.User().Timezone)

	b := v.Set("users", users).
		Set("menu", "settings").
		Render("users")
	html.OK(w, r, b)
}
