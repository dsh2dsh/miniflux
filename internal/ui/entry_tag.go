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
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showTagEntryPage(w http.ResponseWriter, r *http.Request) {
	user, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	tagName, err := url.PathUnescape(request.RouteStringParam(r, "tagName"))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	entryID := request.RouteInt64Param(r, "entryID")

	builder := h.store.NewEntryQueryBuilder(user.ID)
	builder.WithTags([]string{tagName})
	builder.WithEntryID(entryID)
	builder.WithoutStatus(model.EntryStatusRemoved)

	entry, err := builder.GetEntry(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if entry == nil {
		html.NotFound(w, r)
		return
	}

	if entry.ShouldMarkAsReadOnView(user) {
		err = h.store.SetEntriesStatus(r.Context(), user.ID,
			[]int64{entry.ID}, model.EntryStatusRead)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}

		entry.Status = model.EntryStatusRead
	}

	entryPaginationBuilder := h.store.NewEntryPaginationBuilder(user.ID, entry.ID, user.EntryOrder, user.EntryDirection)
	entryPaginationBuilder.WithTags([]string{tagName})
	prevEntry, nextEntry, err := entryPaginationBuilder.Entries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	nextEntryRoute := ""
	if nextEntry != nil {
		nextEntryRoute = route.Path(h.router, "tagEntry", "tagName", url.PathEscape(tagName), "entryID", nextEntry.ID)
	}

	prevEntryRoute := ""
	if prevEntry != nil {
		prevEntryRoute = route.Path(h.router, "tagEntry", "tagName", url.PathEscape(tagName), "entryID", prevEntry.ID)
	}

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("entry", entry)
	view.Set("prevEntry", prevEntry)
	view.Set("nextEntry", nextEntry)
	view.Set("nextEntryRoute", nextEntryRoute)
	view.Set("prevEntryRoute", prevEntryRoute)
	view.Set("user", user)
	view.Set("countUnread", h.store.CountUnreadEntries(r.Context(), user.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), user.ID))
	view.Set("hasSaveEntry", h.store.HasSaveEntry(r.Context(), user.ID))
	html.OK(w, r, view.Render("entry"))
}
