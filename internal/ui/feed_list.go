// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showFeedsPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var feeds model.Feeds
	v.Go(func(ctx context.Context) (err error) {
		feeds, err = h.store.FeedsWithCounters(ctx, v.UserID())
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("feeds", feeds).
		Set("total", len(feeds))
	html.OK(w, r, v.Render("feeds"))
}
