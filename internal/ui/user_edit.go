// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
)

// EditUser shows the form to edit a user.
func (h *handler) showEditUserPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	userID := request.RouteInt64Param(r, "userID")
	var user *model.User
	v.Go(func(ctx context.Context) (err error) {
		user, err = h.store.UserByID(ctx, userID)
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if !v.User().IsAdmin {
		html.Forbidden(w, r)
		return
	} else if user == nil {
		html.NotFound(w, r)
		return
	}

	userForm := &form.UserForm{
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
	}

	v.Set("menu", "settings").
		Set("form", userForm).
		Set("selected_user", user)
	html.OK(w, r, v.Render("edit_user"))
}
