// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"

	"miniflux.app/v2/internal/http/response"
)

func (h *handler) markCategoryAsRead(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	_, err := h.store.MarkCategoryAsRead(r.Context(), userID, id,
		time.Now())
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "categories")
}
