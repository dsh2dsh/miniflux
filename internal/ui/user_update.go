// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"context"
	"errors"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	userID := request.RouteInt64Param(r, "userID")
	var user *model.User
	v.Go(func(ctx context.Context) (err error) {
		user, err = h.store.UserByID(ctx, userID)
		return err
	})

	err := v.Wait()
	switch {
	case err != nil:
		response.ServerError(w, r, err)
		return
	case !v.User().IsAdmin:
		response.Forbidden(w, r)
		return
	case user == nil:
		response.NotFound(w, r)
		return
	case user.ID == v.UserID():
		// Changing your own password must go through the authentication page, which
		// asks for the current password.
		response.BadRequest(w, r, errors.New("you cannot edit yourself"))
		return
	}

	userForm := form.NewUserForm(r)
	v.Set("menu", "settings").
		Set("selected_user", user).
		Set("form", userForm)

	if lerr := userForm.ValidateModification(); lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		response.HTML(w, r, v.Render("edit_user"))
		return
	}

	userMod := model.UserModificationRequest{
		Username: model.OptionalString(userForm.Username),
	}

	lerr := validator.ValidateUserModification(
		r.Context(), h.store, user.ID, &userMod)
	if lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		response.HTML(w, r, v.Render("edit_user"))
		return
	}

	userForm.Merge(user)
	if err = h.store.UpdateUser(r.Context(), user); err != nil {
		response.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "users")
}
