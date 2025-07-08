// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
)

func (h *handler) markFeedAsRead(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	_, err := h.store.MarkFeedAsRead(r.Context(), userID, id, time.Now())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feeds"))
}
