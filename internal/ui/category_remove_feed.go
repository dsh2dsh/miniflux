// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/logging"
)

func (h *handler) removeCategoryFeed(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	categoryID := request.RouteInt64Param(r, "categoryID")

	userID := request.UserID(r)
	exists, err := h.store.CategoryFeedExists(r.Context(), userID, categoryID,
		feedID)
	if err != nil {
		logging.FromContext(r.Context()).Error(
			"storage: unable check feed exists",
			slog.Int64("user_id", userID),
			slog.Int64("category_id", categoryID),
			slog.Int64("feed_id", feedID),
			slog.Any("error", err))
		html.ServerError(w, r, err)
		return
	} else if !exists {
		html.NotFound(w, r)
		return
	}

	err = h.store.RemoveFeed(r.Context(), userID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categoryFeeds", "categoryID",
		categoryID))
}
