// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveUser(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	if !v.User().IsAdmin {
		html.Forbidden(w, r)
		return
	}

	userForm := form.NewUserForm(r)
	v.Set("menu", "settings").
		Set("form", userForm)

	if lerr := userForm.ValidateCreation(); lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_user"))
		return
	}

	if h.store.UserExists(r.Context(), userForm.Username) {
		lerr := locale.NewLocalizedError("error.user_already_exists")
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_user"))
		return
	}

	userCreationRequest := &model.UserCreationRequest{
		Username: userForm.Username,
		Password: userForm.Password,
		IsAdmin:  userForm.IsAdmin,
	}

	lerr := validator.ValidateUserCreationWithPassword(
		r.Context(), h.store, userCreationRequest)
	if lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_user"))
		return
	}

	_, err := h.store.CreateUser(r.Context(), userCreationRequest)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "users"))
}
