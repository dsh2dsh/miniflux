// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) showFeedsPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	feeds, err := h.store.FeedsWithCounters(r.Context(), v.User().ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("feeds", feeds).
		Set("total", len(feeds))
	html.OK(w, r, v.Render("feeds"))
}
