// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showCreateUserPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if !v.User().IsAdmin {
		html.Forbidden(w, r)
		return
	}

	v.Set("form", &form.UserForm{}).
		Set("menu", "settings")
	html.OK(w, r, v.Render("create_user"))
}
