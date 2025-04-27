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
	loggedUser, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if !loggedUser.IsAdmin {
		html.Forbidden(w, r)
		return
	}

	selectedUserID := request.RouteInt64Param(r, "userID")
	selectedUser, err := h.store.UserByID(r.Context(), selectedUserID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if selectedUser == nil {
		html.NotFound(w, r)
		return
	}

	if selectedUser.ID == loggedUser.ID {
		html.BadRequest(w, r, errors.New("you cannot remove yourself"))
		return
	}

	err = h.store.RemoveUser(r.Context(), selectedUser.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "users"))
}
