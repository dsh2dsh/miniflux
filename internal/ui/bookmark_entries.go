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

func (h *handler) showStarredPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user := v.User()

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(user.ID).
		WithoutStatus(model.EntryStatusRemoved).
		WithStarred(true).
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "starred").
		Set("total", count).
		Set("entries", entries).
		Set("pagination", getPagination(route.Path(h.router, "starred"),
			count, offset, user.EntriesPerPage))
	html.OK(w, r, v.Render("bookmark_entries"))
}
