// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showAddSubscriptionPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	categories, err := h.store.Categories(r.Context(), v.User().ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("categories", categories).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent()).
		Set("form", &form.SubscriptionForm{CategoryID: 0}).
		Set("hasProxyConfigured", config.Opts.HasHTTPClientProxyURLConfigured())
	html.OK(w, r, v.Render("add_subscription"))
}
