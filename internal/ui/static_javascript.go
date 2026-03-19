// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"

	"miniflux.app/v2/internal/ui/static"
)

func (h *handler) showJavascript(w http.ResponseWriter, r *http.Request) {
	filename := request.RouteStringParam(r, "name")
	compressed := static.JavascriptBundle(filename)
	if compressed == nil {
		response.NotFound(w, r)
		return
	}

	response.New(w, r).WithoutCompression().
		WithHeader("Content-Encoding", "gzip").
		WithLongCaching().
		WithHeader("Content-Type", "text/javascript; charset=utf-8").
		WithBodyAsBytes(compressed).
		Write()
}
