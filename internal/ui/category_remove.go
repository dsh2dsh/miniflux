// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
)

func (h *handler) removeCategory(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	affected, err := h.store.RemoveCategory(r.Context(), userID, id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if !affected {
		html.NotFound(w, r)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "categories"))
}
