// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"net/url"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showTagEntriesAllPage(w http.ResponseWriter,
	r *http.Request,
) {
	tagName, err := url.PathUnescape(request.RouteStringParam(r, "tagName"))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v := h.View(r).WithSaveEntry()
	user, err := v.WaitUser()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(v.UserID()).
		WithoutStatus(model.EntryStatusRemoved).
		WithTags([]string{tagName}).
		WithSorting("status", "asc").
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("tagName", tagName).
		Set("total", count).
		Set("entries", entries).
		Set("pagination", getPagination(
			route.Path(h.router, "tagEntriesAll", "tagName", url.PathEscape(tagName)),
			count, offset, user.EntriesPerPage)).
		Set("showOnlyUnreadEntries", false)
	html.OK(w, r, v.Render("tag_entries"))
}
