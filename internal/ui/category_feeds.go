// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showCategoryFeedsPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	categoryID := request.RouteInt64Param(r, "categoryID")
	category, err := h.store.Category(r.Context(), v.User().ID, categoryID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	feeds, err := h.store.FeedsByCategoryWithCounters(r.Context(), v.User().ID,
		categoryID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "categories").
		Set("category", category).
		Set("feeds", feeds).
		Set("total", len(feeds))
	html.OK(w, r, v.Render("category_feeds"))
}
