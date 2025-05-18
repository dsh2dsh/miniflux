// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) sharedEntries(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	builder := h.store.NewEntryQueryBuilder(v.User().ID).
		WithShareCodeNotEmpty().
		WithSorting(v.User().EntryOrder, v.User().EntryDirection).
		WithSorting("id", v.User().EntryDirection).
		WithOffset(offset).
		WithLimit(v.User().EntriesPerPage)

	g, ctx := errgroup.WithContext(r.Context())
	var entries model.Entries
	g.Go(func() (err error) {
		entries, err = builder.GetEntries(ctx)
		return
	})

	var count int
	g.Go(func() (err error) {
		count, err = builder.CountEntries(ctx)
		return
	})

	if err := g.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "history").
		Set("entries", entries).
		Set("total", count).
		Set("pagination", getPagination(route.Path(h.router, "sharedEntries"),
			count, offset, v.User().EntriesPerPage))
	html.OK(w, r, v.Render("shared_entries"))
}
