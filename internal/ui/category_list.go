// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showCategoryListPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var categories []model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.CategoriesWithFeedCount(ctx, v.UserID())
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "categories").
		Set("categories", categories).
		Set("total", len(categories))
	html.OK(w, r, v.Render("categories"))
}
