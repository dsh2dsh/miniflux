// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showFeedIcon(w http.ResponseWriter, r *http.Request) {
	id := request.RouteStringParam(r, "externalIconID")
	icon, err := h.store.IconByExternalID(r.Context(), id)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if icon == nil {
		html.NotFound(w, r)
		return
	}

	resp := response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Security-Policy",
			response.ContentSecurityPolicyForUntrustedContent).
		WithHeader("Content-Type", icon.MimeType).
		WithBody(icon.Content)

	if icon.MimeType != "image/svg+xml" {
		resp.WithoutCompression()
	}
	resp.Write()
}
