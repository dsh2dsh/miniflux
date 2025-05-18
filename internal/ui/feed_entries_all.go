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

func (h *handler) showFeedEntriesAllPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	feedID := request.RouteInt64Param(r, "feedID")
	feed, err := h.store.FeedByID(r.Context(), v.User().ID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if feed == nil {
		html.NotFound(w, r)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	builder := h.store.NewEntryQueryBuilder(v.User().ID).
		WithFeedID(feed.ID).
		WithoutStatus(model.EntryStatusRemoved).
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

	v.Set("menu", "feeds").
		Set("feed", feed).
		Set("entries", entries).
		Set("total", count).
		Set("pagination", getPagination(
			route.Path(h.router, "feedEntriesAll", "feedID", feed.ID), count, offset,
			v.User().EntriesPerPage)).
		Set("showOnlyUnreadEntries", false)
	html.OK(w, r, v.Render("feed_entries"))
}
