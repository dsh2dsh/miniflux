// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response"
)

func (h *handler) showImportPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		response.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds")
	response.HTML(w, r, v.Render("import"))
}
