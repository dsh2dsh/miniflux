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

func (h *handler) showSearchPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	searchQuery := request.QueryStringParam(r, "q", "")
	offset := request.QueryIntParam(r, "offset", 0)

	var entries model.Entries
	var entriesCount int

	if searchQuery != "" {
		builder := h.store.NewEntryQueryBuilder(v.User().ID).
			WithSearchQuery(searchQuery).
			WithoutStatus(model.EntryStatusRemoved).
			WithOffset(offset).
			WithLimit(v.User().EntriesPerPage)

		g, ctx := errgroup.WithContext(r.Context())
		g.Go(func() (err error) {
			entries, err = builder.GetEntries(ctx)
			return
		})

		g.Go(func() (err error) {
			entriesCount, err = builder.CountEntries(ctx)
			return
		})

		if err := g.Wait(); err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	pagination := getPagination(route.Path(h.router, "search"), entriesCount,
		offset, v.User().EntriesPerPage)
	pagination.SearchQuery = searchQuery

	v.Set("menu", "search").
		Set("searchQuery", searchQuery).
		Set("entries", entries).
		Set("total", entriesCount).
		Set("pagination", pagination)
	html.OK(w, r, v.Render("search"))
}
