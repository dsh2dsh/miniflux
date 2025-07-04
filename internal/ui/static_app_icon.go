// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"path/filepath"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/static"
)

func (h *handler) showBinaryFile(w http.ResponseWriter, r *http.Request) {
	filename := request.RouteStringParam(r, "filename")
	blob, err := static.LoadBinaryFile(filename)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	resp := response.New(w, r).WithLongCaching().WithBody(blob)
	switch filepath.Ext(filename) {
	case ".png":
		resp.WithoutCompression().
			WithHeader("Content-Type", "image/png")
	case ".svg":
		resp.WithHeader("Content-Type", "image/svg+xml")
	}
	resp.Write()
}
