// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showAPIKeysPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	user, err := v.WaitUser()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	var keys []*model.APIKey
	v.Go(func(ctx context.Context) (err error) {
		keys, err = h.store.APIKeys(ctx, user.ID)
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("apiKeys", keys)
	html.OK(w, r, v.Render("api_keys"))
}
