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

func (h *handler) showSearchEntryPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	entryID := request.RouteInt64Param(r, "entryID")
	searchQuery := request.QueryStringParam(r, "q", "")
	entry, err := h.store.NewEntryQueryBuilder(v.User().ID).
		WithSearchQuery(searchQuery).
		WithEntryID(entryID).
		WithoutStatus(model.EntryStatusRemoved).
		GetEntry(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if entry == nil {
		html.NotFound(w, r)
		return
	}

	if entry.ShouldMarkAsReadOnView(v.User()) {
		err = h.store.SetEntriesStatus(r.Context(), v.User().ID,
			[]int64{entry.ID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		entry.Status = model.EntryStatusRead
	}

	prevEntry, nextEntry, err := h.store.NewEntryPaginationBuilder(
		v.User().ID, entry.ID, v.User().EntryOrder, v.User().EntryDirection).
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
