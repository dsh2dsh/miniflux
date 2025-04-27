// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) showCategoryEntriesStarredPage(w http.ResponseWriter, r *http.Request) {
	user, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	categoryID := request.RouteInt64Param(r, "categoryID")
	category, err := h.store.Category(r.Context(), request.UserID(r), categoryID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if category == nil {
		html.NotFound(w, r)
		return
	}

	offset := request.QueryIntParam(r, "offset", 0)
	builder := h.store.NewEntryQueryBuilder(user.ID)
	builder.WithCategoryID(category.ID)
	builder.WithSorting(user.EntryOrder, user.EntryDirection)
	builder.WithSorting("id", user.EntryDirection)
	builder.WithoutStatus(model.EntryStatusRemoved)
	builder.WithStarred(true)
	builder.WithOffset(offset)
	builder.WithLimit(user.EntriesPerPage)

	entries, err := builder.GetEntries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	count, err := builder.CountEntries(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("category", category)
	view.Set("total", count)
	view.Set("entries", entries)
	view.Set("pagination", getPagination(route.Path(h.router, "categoryEntriesStarred", "categoryID", category.ID), count, offset, user.EntriesPerPage))
	view.Set("menu", "categories")
	view.Set("user", user)
	view.Set("countUnread", h.store.CountUnreadEntries(r.Context(), user.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), user.ID))
	view.Set("hasSaveEntry", h.store.HasSaveEntry(r.Context(), user.ID))
	view.Set("showOnlyStarredEntries", true)
	html.OK(w, r, view.Render("category_entries"))
}
