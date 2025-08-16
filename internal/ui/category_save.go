// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveCategory(w http.ResponseWriter, r *http.Request) {
	f := form.NewCategoryForm(r)
	createRequest := model.CategoryCreationRequest{Title: f.Title}

	userID := request.UserID(r)
	lerr := validator.ValidateCategoryCreation(r.Context(), h.store, userID,
		&createRequest)
	if lerr == nil {
		_, err := h.store.CreateCategory(r.Context(), userID, &createRequest)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		h.redirect(w, r, "categories")
		return
	}

	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "categories").
		Set("form", f).
		Set("errorMessage", lerr.Translate(v.User().Language))
	html.OK(w, r, v.Render("create_category"))
}
