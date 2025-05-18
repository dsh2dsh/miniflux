// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"
	"net/url"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showTagEntriesAllPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	tagName, err := url.PathUnescape(request.RouteStringParam(r, "tagName"))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	builder := h.store.NewEntryQueryBuilder(v.User().ID).
		WithoutStatus(model.EntryStatusRemoved).
		WithTags([]string{tagName}).
		WithSorting("status", "asc").
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

	v.Set("tagName", tagName).
		Set("total", count).
		Set("entries", entries).
		Set("pagination", getPagination(
			route.Path(h.router, "tagEntriesAll", "tagName", url.PathEscape(tagName)),
			count, offset, v.User().EntriesPerPage)).
		Set("showOnlyUnreadEntries", false)
	html.OK(w, r, v.Render("tag_entries"))
}
