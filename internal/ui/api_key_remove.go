// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"errors"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
)

func (h *handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "keyID")
	affected, err := h.store.DeleteAPIKey(r.Context(), request.UserID(r), id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if !affected {
		html.ServerError(w, r, errors.New("API Key not found"))
		return
	}
	html.Redirect(w, r, route.Path(h.router, "apiKeys"))
}
