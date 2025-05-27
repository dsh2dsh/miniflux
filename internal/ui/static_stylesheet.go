// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/static"
)

func (h *handler) showStylesheet(w http.ResponseWriter, r *http.Request) {
	filename := request.RouteStringParam(r, "name")
	b, found := static.StylesheetBundles[filename]
	if !found {
		html.NotFound(w, r)
		return
	}

	response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Type", "text/css; charset=utf-8").
		WithBody(b).
		Write()
}
