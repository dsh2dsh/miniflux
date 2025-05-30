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

func (h *handler) showHistoryPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user := v.User()

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(v.UserID()).
		WithStatus(model.EntryStatusRead).
		WithSorting("changed_at", "DESC").
		WithSorting("published_at", "DESC").
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "history").
		Set("entries", entries).
		Set("total", count).
		Set("pagination", getPagination(route.Path(h.router, "history"),
			count, offset, user.EntriesPerPage))
	html.OK(w, r, v.Render("history_entries"))
}
