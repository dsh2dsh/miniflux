// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/static"
)

func (h *handler) showFavicon(w http.ResponseWriter, r *http.Request) {
	etag, err := static.GetBinaryFileChecksum("favicon.ico")
	if err != nil {
		html.NotFound(w, r)
		return
	}

	response.New(w, r).WithCaching(etag, 48*time.Hour, func(b *response.Builder) {
		f, err := static.OpenBinaryFile("favicon.ico")
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		defer f.Close()

		b.WithHeader("Content-Type", "image/x-icon")
		b.WithoutCompression()
		b.WithBody(f)
		b.Write()
	})
}
