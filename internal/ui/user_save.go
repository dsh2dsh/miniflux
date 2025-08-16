// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveUser(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	if !user.IsAdmin {
		html.Forbidden(w, r)
		return
	}

	f := form.NewUserForm(r)
	if lerr := f.ValidateCreation(); lerr != nil {
		h.showSaveUserError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("create_user"))
		})
		return
	}

	if h.store.UserExists(r.Context(), f.Username) {
		h.showSaveUserError(w, r, func(v *View) {
			lerr := locale.NewLocalizedError("error.user_already_exists")
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("create_user"))
		})
		return
	}

	createRequest := model.UserCreationRequest{
		Username: f.Username,
		Password: f.Password,
		IsAdmin:  f.IsAdmin,
	}

	lerr := validator.ValidateUserCreationWithPassword(r.Context(), h.store,
		&createRequest)
	if lerr != nil {
		h.showSaveUserError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("create_user"))
		})
		return
	}

	_, err := h.store.CreateUser(r.Context(), &createRequest)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "users")
}

func (h *handler) showSaveUserError(w http.ResponseWriter, r *http.Request,
	renderFunc func(v *View),
) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if !v.User().IsAdmin {
		html.Forbidden(w, r)
		return
	}

	v.Set("menu", "settings")
	renderFunc(v)
}
