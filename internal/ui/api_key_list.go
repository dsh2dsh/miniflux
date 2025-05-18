// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showAPIKeysPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	apiKeys, err := h.store.APIKeys(r.Context(), v.User().ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("apiKeys", apiKeys)
	html.OK(w, r, v.Render("api_keys"))
}
