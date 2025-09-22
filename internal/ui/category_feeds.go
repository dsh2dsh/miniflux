// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showCategoryFeedsPage(w http.ResponseWriter, r *http.Request,
) {
	v := h.View(r)

	id := request.RouteInt64Param(r, "categoryID")
	var category *model.Category
	v.Go(func(ctx context.Context) (err error) {
		category, err = h.store.Category(ctx, v.UserID(), id)
		return err
	})

	var feeds model.Feeds
	v.Go(func(ctx context.Context) (err error) {
		feeds, err = h.store.FeedsByCategoryWithCounters(ctx, v.UserID(), id)
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	v.Set("menu", "categories").
		Set("category", category).
		Set("feeds", feeds).
		Set("total", len(feeds))
	html.OK(w, r, v.Render("category_feeds"))
}
