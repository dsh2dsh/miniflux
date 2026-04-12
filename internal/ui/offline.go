// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showOfflinePage(w http.ResponseWriter, r *http.Request) {
	response.HTML(w, r, view.New(h.tpl, r).Render("offline"))
}
