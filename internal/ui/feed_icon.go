// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) showFeedIcon(w http.ResponseWriter, r *http.Request) {
	id := request.RouteStringParam(r, "externalIconID")
	icon, err := h.store.IconByExternalID(r.Context(), id)
	if err != nil {
		response.ServerError(w, r, err)
		return
	} else if icon == nil {
		response.NotFound(w, r)
		return
	}

	resp := response.New(w, r).
		WithLongCaching().
		WithHeader("Content-Security-Policy",
			response.ContentSecurityPolicyForUntrustedContent).
		WithHeader("Content-Type", icon.MimeType).
		WithBodyAsBytes(icon.Content)

	if icon.MimeType != "image/svg+xml" {
		resp.WithoutCompression()
	}
	resp.Write()
}
