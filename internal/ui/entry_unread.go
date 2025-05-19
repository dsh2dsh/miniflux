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

func (h *handler) showUnreadEntryPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()

	entryID := request.RouteInt64Param(r, "entryID")
	var entry *model.Entry
	v.Go(func(ctx context.Context) (err error) {
		entry, err = h.store.NewEntryQueryBuilder(v.UserID()).
			WithEntryID(entryID).
			WithoutStatus(model.EntryStatusRemoved).
			GetEntry(ctx)
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if entry == nil {
		html.Redirect(w, r, route.Path(h.router, "unread"))
		return
	}

	// Make sure we always get the pagination in unread mode even if the page is
	// refreshed.
	if entry.Status == model.EntryStatusRead {
		err := h.store.SetEntriesStatus(r.Context(), v.UserID(),
			[]int64{entry.ID}, model.EntryStatusUnread)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
	}

	prevEntry, nextEntry, err := h.store.NewEntryPaginationBuilder(
		v.UserID(), entryID, v.User().EntryOrder, v.User().EntryDirection).
		WithStatus(model.EntryStatusUnread).
		WithGloballyVisible().
		Entries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	var nextEntryRoute string
	if nextEntry != nil {
		nextEntryRoute = route.Path(h.router, "unreadEntry", "entryID",
			nextEntry.ID)
	}

	var prevEntryRoute string
	if prevEntry != nil {
		prevEntryRoute = route.Path(h.router, "unreadEntry", "entryID",
			prevEntry.ID)
	}

	if entry.ShouldMarkAsReadOnView(v.User()) {
		entry.Status = model.EntryStatusRead
	}

	// Restore entry read status if needed after fetching the pagination.
	if entry.Status == model.EntryStatusRead {
		err := h.store.SetEntriesStatus(r.Context(), v.User().ID,
			[]int64{entry.ID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		if v.CountUnread() > 0 {
			v.Set("countUnread", v.CountUnread()-1)
		}
	}

	v.Set("menu", "unread").
		Set("entry", entry).
		Set("prevEntry", prevEntry).
		Set("nextEntry", nextEntry).
		Set("nextEntryRoute", nextEntryRoute).
		Set("prevEntryRoute", prevEntryRoute)
	html.OK(w, r, v.Render("entry"))
}
