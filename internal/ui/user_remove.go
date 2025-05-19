// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"errors"
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) removeUser(w http.ResponseWriter, r *http.Request) {
	g, ctx := errgroup.WithContext(r.Context())

	var user *model.User
	g.Go(func() (err error) {
		user, err = h.store.UserByID(ctx, request.UserID(r))
		return
	})

	selectedUserID := request.RouteInt64Param(r, "userID")
	var selectedUser *model.User
	g.Go(func() (err error) {
		selectedUser, err = h.store.UserByID(ctx, selectedUserID)
		return
	})

	if err := g.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if !user.IsAdmin {
		html.Forbidden(w, r)
		return
	} else if selectedUser == nil {
		html.NotFound(w, r)
		return
	}

	if selectedUser.ID == user.ID {
		html.BadRequest(w, r, errors.New("you cannot remove yourself"))
		return
	}

	err := h.store.RemoveUser(r.Context(), selectedUser.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "users"))
}
