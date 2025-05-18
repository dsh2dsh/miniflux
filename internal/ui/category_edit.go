// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showEditCategoryPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	category, err := h.store.Category(r.Context(), v.User().ID,
		request.RouteInt64Param(r, "categoryID"))
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	categoryForm := form.CategoryForm{
		Title:        category.Title,
		HideGlobally: category.HideGlobally,
	}

	v.Set("menu", "categories").
		Set("form", categoryForm).
		Set("category", category)
	html.OK(w, r, v.Render("edit_category"))
}
