// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) removeCategoryFeed(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	categoryID := request.RouteInt64Param(r, "categoryID")

	userID := request.UserID(r)
	exists, err := h.store.CategoryFeedExists(r.Context(), userID, categoryID,
		feedID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if !exists {
		response.NotFound(w, r)
		return
	}

	affected, err := h.store.RemoveFeed(r.Context(), userID, feedID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if !affected {
		response.NotFound(w, r)
		return
	}
	h.redirect(w, r, "categoryFeeds", "categoryID", categoryID)
}
