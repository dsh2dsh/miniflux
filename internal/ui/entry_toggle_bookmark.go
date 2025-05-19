// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
)

func (h *handler) toggleBookmark(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "entryID")
	err := h.store.ToggleBookmark(r.Context(), request.UserID(r), id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, "OK")
}
