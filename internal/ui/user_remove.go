// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"errors"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) removeUser(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	userID := request.RouteInt64Param(r, "userID")
	if !user.IsAdmin {
		response.Forbidden(w, r)
		return
	} else if userID == user.ID {
		response.BadRequest(w, r, errors.New("you cannot remove yourself"))
		return
	}

	affected, err := h.store.RemoveUser(r.Context(), userID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if !affected {
		response.NotFound(w, r)
		return
	}
	h.redirect(w, r, "users")
}
