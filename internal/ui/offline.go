// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showOfflinePage(w http.ResponseWriter, r *http.Request) {
	view := view.New(h.tpl, r, nil)
	html.OK(w, r, view.Render("offline"))
}
