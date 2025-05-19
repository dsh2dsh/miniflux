// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showSearchPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user, err := v.WaitUser()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	searchQuery := request.QueryStringParam(r, "q", "")
	offset := request.QueryIntParam(r, "offset", 0)
	var entries model.Entries
	var count int

	if searchQuery != "" {
		query := h.store.NewEntryQueryBuilder(v.UserID()).
			WithSearchQuery(searchQuery).
			WithoutStatus(model.EntryStatusRemoved).
			WithOffset(offset).
			WithLimit(user.EntriesPerPage)

		entries, count, err = v.WaitEntriesCount(query)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	} else if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	pagination := getPagination(route.Path(h.router, "search"), count,
		offset, user.EntriesPerPage)
	pagination.SearchQuery = searchQuery

	v.Set("menu", "search").
		Set("searchQuery", searchQuery).
		Set("entries", entries).
		Set("total", count).
		Set("pagination", pagination)
	html.OK(w, r, v.Render("search"))
}
