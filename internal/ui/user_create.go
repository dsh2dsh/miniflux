// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showCreateUserPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		response.ServerError(w, r, err)
		return
	} else if !v.User().IsAdmin {
		response.Forbidden(w, r)
		return
	}

	v.Set("form", &form.UserForm{}).
		Set("menu", "settings")
	response.HTML(w, r, v.Render("create_user"))
}
