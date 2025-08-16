// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) removeSession(w http.ResponseWriter, r *http.Request) {
	id := request.RouteStringParam(r, "sessionID")
	err := h.store.RemoveAppSessionByID(r.Context(), id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "sessions")
}
