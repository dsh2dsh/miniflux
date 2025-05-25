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

func (h *handler) showUnreadCategoryEntryPage(w http.ResponseWriter,
	r *http.Request,
) {
	v := h.View(r).WithSaveEntry()

	categoryID := request.RouteInt64Param(r, "categoryID")
	entryID := request.RouteInt64Param(r, "entryID")
	var entry *model.Entry
	v.Go(func(ctx context.Context) (err error) {
		entry, err = h.store.NewEntryQueryBuilder(v.UserID()).
			WithCategoryID(categoryID).
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
			[]int64{entryID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		entry.Status = model.EntryStatusRead
	}

	pb := h.store.NewEntryPaginationBuilder(v.UserID(), entryID,
		v.User().EntryOrder, v.User().EntryDirection).
		WithCategoryID(categoryID).
		WithStatus(model.EntryStatusUnread)

	if entry.Status == model.EntryStatusRead {
		err := h.store.SetEntriesStatus(r.Context(), v.UserID(),
			[]int64{entryID}, model.EntryStatusUnread)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	prevEntry, nextEntry, err := pb.Entries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	var nextEntryRoute string
	if nextEntry != nil {
		nextEntryRoute = route.Path(h.router, "unreadCategoryEntry",
			"categoryID", categoryID, "entryID", nextEntry.ID)
	}

	var prevEntryRoute string
	if prevEntry != nil {
		prevEntryRoute = route.Path(h.router, "unreadCategoryEntry",
			"categoryID", categoryID, "entryID", prevEntry.ID)
	}

	// Restore entry read status if needed after fetching the pagination.
	if entry.Status == model.EntryStatusRead {
		err := h.store.SetEntriesStatus(r.Context(), v.UserID(),
			[]int64{entryID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	if v.User().AlwaysOpenExternalLinks() {
		html.Redirect(w, r, entry.URL)
		return
	}

	b := v.Set("menu", "categories").
		Set("entry", entry).
		Set("prevEntry", prevEntry).
		Set("nextEntry", nextEntry).
		Set("nextEntryRoute", nextEntryRoute).
		Set("prevEntryRoute", prevEntryRoute).
		Render("entry")
	html.OK(w, r, b)
}
