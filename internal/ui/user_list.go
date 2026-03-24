// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showUsersPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var users model.Users
	v.Go(func(ctx context.Context) (err error) {
		users, err = h.store.Users(ctx)
		return err
	})

	if err := v.Wait(); err != nil {
		response.ServerError(w, r, err)
		return
	} else if !v.User().Operator() {
		response.Forbidden(w, r)
		return
	}

	users.UseTimezone(v.User().Timezone)
	v.Set("users", users).
		Set("menu", "settings")
	response.HTML(w, r, v.Render("users"))
}
