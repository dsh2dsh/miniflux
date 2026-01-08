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
	user := v.User()

	searchQuery := request.QueryStringParam(r, "q", "")
	unreadOnly := request.QueryBoolParam(r, "unread", false)
	offset := request.QueryIntParam(r, "offset", 0)
	var entries model.Entries
	var count int

	if searchQuery != "" {
		query := h.store.NewEntryQueryBuilder(v.UserID()).
			WithSearchQuery(searchQuery).
			WithOffset(offset).
			WithLimit(user.EntriesPerPage)

		if unreadOnly {
			query.WithStatus(model.EntryStatusUnread)
		} else {
			query.WithoutStatus(model.EntryStatusRemoved)
		}

		var err error
		entries, count, err = v.WaitEntriesCount(query)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	} else if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	pagination := getPagination(
		route.Path(h.router, "search"), count, offset, user.EntriesPerPage).
		WithSearchQuery(searchQuery).
		WithUnreadOnly(unreadOnly)

	v.Set("menu", "search").
		Set("searchQuery", searchQuery).
		Set("entries", entries).
		Set("total", count).
		Set("pagination", pagination).
		Set("showOnlyUnreadEntries", unreadOnly)
	html.OK(w, r, v.Render("search"))
}
