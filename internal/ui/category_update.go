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
	f := form.NewCategoryForm(r)
	modifyRequest := model.CategoryModificationRequest{
		Title:        model.SetOptionalField(f.Title),
		HideGlobally: model.SetOptionalField(f.HideGlobally),
	}
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	lerr := validator.ValidateCategoryModification(r.Context(), h.store, userID,
		id, &modifyRequest)
	if lerr != nil {
		h.showUpdateCategoryError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("create_category"))
		})
		return
	}

	category, err := h.store.Category(r.Context(), userID, id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	modifyRequest.Patch(category)
	affected, err := h.store.UpdateCategory(r.Context(), category)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if !affected {
		html.NotFound(w, r)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categoryFeeds", "categoryID", id))
}

func (h *handler) showUpdateCategoryError(w http.ResponseWriter,
	r *http.Request, renderFunc func(v *View),
) {
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

	v.Set("menu", "categories").
		Set("category", category)
	renderFunc(v)
}
