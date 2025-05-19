// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) markCategoryAsRead(w http.ResponseWriter, r *http.Request) {
	g, ctx := errgroup.WithContext(r.Context())
	userID := request.UserID(r)
	categoryID := request.RouteInt64Param(r, "categoryID")

	var category *model.Category
	g.Go(func() (err error) {
		category, err = h.store.Category(ctx, userID, categoryID)
		return
	})

	g.Go(func() error {
		return h.store.MarkCategoryAsRead(ctx, userID, categoryID, time.Now())
	})

	if err := g.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categories"))
}
