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

func (h *handler) showSearchEntryPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()

	entryID := request.RouteInt64Param(r, "entryID")
	searchQuery := request.QueryStringParam(r, "q", "")
	var entry *model.Entry
	v.Go(func(ctx context.Context) (err error) {
		entry, err = h.store.NewEntryQueryBuilder(v.UserID()).
			WithSearchQuery(searchQuery).
			WithEntryID(entryID).
			WithoutStatus(model.EntryStatusRemoved).
			GetEntry(ctx)
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if entry == nil {
		html.NotFound(w, r)
		return
	}

	if entry.ShouldMarkAsReadOnView(v.User()) {
		err := h.store.SetEntriesStatus(r.Context(), v.UserID(),
			[]int64{entry.ID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		entry.Status = model.EntryStatusRead
	}

	prevEntry, nextEntry, err := h.store.NewEntryPaginationBuilder(
		v.UserID(), entryID, v.User().EntryOrder, v.User().EntryDirection).
		WithSearchQuery(searchQuery).
		Entries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	var nextEntryRoute string
	if nextEntry != nil {
		nextEntryRoute = route.Path(h.router, "searchEntry", "entryID",
			nextEntry.ID)
	}

	var prevEntryRoute string
	if prevEntry != nil {
		prevEntryRoute = route.Path(h.router, "searchEntry", "entryID",
			prevEntry.ID)
	}

	v.Set("menu", "search").
		Set("searchQuery", searchQuery).
		Set("entry", entry).
		Set("prevEntry", prevEntry).
		Set("nextEntry", nextEntry).
		Set("nextEntryRoute", nextEntryRoute).
		Set("prevEntryRoute", prevEntryRoute)
	html.OK(w, r, v.Render("entry"))
}
