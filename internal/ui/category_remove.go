// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) removeCategory(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	affected, err := h.store.RemoveCategory(r.Context(), userID, id)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if !affected {
		response.NotFound(w, r)
		return
	}
	h.redirect(w, r, "categories")
}
