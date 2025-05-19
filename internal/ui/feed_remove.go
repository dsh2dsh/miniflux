// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
)

func (h *handler) removeFeed(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	feedID := request.RouteInt64Param(r, "feedID")

	if !h.store.FeedExists(r.Context(), userID, feedID) {
		html.NotFound(w, r)
		return
	}

	err := h.store.RemoveFeed(r.Context(), userID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feeds"))
}
