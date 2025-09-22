// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	userID := request.RouteInt64Param(r, "userID")
	var user *model.User
	v.Go(func(ctx context.Context) (err error) {
		user, err = h.store.UserByID(ctx, userID)
		return err
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

	userForm := form.NewUserForm(r)
	v.Set("menu", "settings").
		Set("selected_user", user).
		Set("form", userForm)

	if lerr := userForm.ValidateModification(); lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("edit_user"))
		return
	}

	alreadyExists := h.store.AnotherUserExists(r.Context(), userID,
		userForm.Username)
	if alreadyExists {
		lerr := locale.NewLocalizedError("error.user_already_exists")
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("edit_user"))
		return
	}

	userForm.Merge(user)
	err := h.store.UpdateUser(r.Context(), user)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "users")
}
