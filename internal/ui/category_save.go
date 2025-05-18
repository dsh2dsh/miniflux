// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveCategory(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	categoryForm := form.NewCategoryForm(r)
	creationRequest := &model.CategoryCreationRequest{
		Title: categoryForm.Title,
	}

	lerr := validator.ValidateCategoryCreation(r.Context(), h.store, v.User().ID,
		creationRequest)
	if lerr != nil {
		v.Set("menu", "categories").
			Set("form", categoryForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_category"))
		return
	}

	_, err := h.store.CreateCategory(r.Context(), v.User().ID, creationRequest)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categories"))
}
