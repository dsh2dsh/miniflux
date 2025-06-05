// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"errors"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
)

func (h *handler) removeUser(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	userID := request.RouteInt64Param(r, "userID")
	if !user.IsAdmin {
		html.Forbidden(w, r)
		return
	} else if userID == user.ID {
		html.BadRequest(w, r, errors.New("you cannot remove yourself"))
		return
	}

	affected, err := h.store.RemoveUser(r.Context(), userID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if !affected {
		html.NotFound(w, r)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "users"))
}
