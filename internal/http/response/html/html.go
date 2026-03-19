// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package html // import "miniflux.app/v2/internal/http/response/html"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response"
)

// OK creates a new HTML response with a 200 status code.
func OK(w http.ResponseWriter, r *http.Request, body []byte) {
	response.New(w, r).
		WithHeader("Content-Type", "text/html; charset=utf-8").
		WithHeader("Cache-Control",
			"no-cache, max-age=0, must-revalidate, no-store").
		WithBodyAsBytes(body).
		Write()
}
