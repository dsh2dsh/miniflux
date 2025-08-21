// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showFeedEntriesPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user := v.User()

	feedID := request.RouteInt64Param(r, "feedID")
	var feed *model.Feed
	v.Go(func(ctx context.Context) (err error) {
		feed, err = h.store.FeedByID(ctx, v.UserID(), feedID)
		return
	})

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(v.UserID()).
		WithFeedID(feedID).
		WithStatus(model.EntryStatusUnread).
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if feed == nil {
		html.NotFound(w, r)
		return
	}

	v.Set("menu", "feeds").
		Set("feed", feed).
		Set("entries", entries).
		Set("total", count).
		Set("pagination", getPagination(
			route.Path(h.router, "feedEntries", "feedID", feedID), count, offset,
			user.EntriesPerPage)).
		Set("markReadOnScroll", user.MarkReadOnScroll()).
		Set("showOnlyUnreadEntries", true)
	html.OK(w, r, v.Render("feed_entries"))
}
