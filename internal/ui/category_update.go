// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	id := request.RouteInt64Param(r, "categoryID")
	var category *model.Category
	v.Go(func(ctx context.Context) (err error) {
		category, err = h.store.Category(ctx, v.UserID(), id)
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	categoryForm := form.NewCategoryForm(r)
	categoryRequest := &model.CategoryModificationRequest{
		Title:        model.SetOptionalField(categoryForm.Title),
		HideGlobally: model.SetOptionalField(categoryForm.HideGlobally),
	}

	lerr := validator.ValidateCategoryModification(r.Context(), h.store,
		v.UserID(), id, categoryRequest)
	if lerr != nil {
		v.Set("menu", "categories").
			Set("form", categoryForm).
			Set("category", category).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_category"))
		return
	}

	categoryRequest.Patch(category)
	err := h.store.UpdateCategory(r.Context(), category)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categoryFeeds", "categoryID", id))
}
