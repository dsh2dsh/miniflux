// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showLoginPage(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	if user != nil {
		html.Redirect(w, r, route.Path(h.router, user.DefaultHomePage))
		return
	}

	v := view.New(h.tpl, r, nil)
	html.OK(w, r, v.Render("login"))
}
