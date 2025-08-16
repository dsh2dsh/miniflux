// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/cookie"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
)

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := request.Session(r)
	if err := h.store.RemoveAppSessionByID(ctx, sess.ID); err != nil {
		html.ServerError(w, r, err)
		return
	}

	http.SetCookie(w, cookie.ExpiredSession())
	h.redirect(w, r, "login")
}
