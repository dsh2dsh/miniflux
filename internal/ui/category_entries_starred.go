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

func (h *handler) showCategoryEntriesStarredPage(w http.ResponseWriter,
	r *http.Request,
) {
	v := h.View(r).WithSaveEntry()
	user := v.User()

	id := request.RouteInt64Param(r, "categoryID")
	var category *model.Category
	v.Go(func(ctx context.Context) (err error) {
		category, err = h.store.Category(ctx, v.UserID(), id)
		return err
	})

	offset := request.QueryIntParam(r, "offset", 0)
	query := h.store.NewEntryQueryBuilder(user.ID).
		WithCategoryID(id).
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithoutStatus(model.EntryStatusRemoved).
		WithStarred(true).
		WithOffset(offset).
		WithLimit(user.EntriesPerPage)

	entries, count, err := v.WaitEntriesCount(query)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if category == nil {
		html.NotFound(w, r)
		return
	}

	v.Set("menu", "categories").
		Set("category", category).
		Set("total", count).
		Set("entries", entries).
		Set("pagination", getPagination(
			route.Path(h.router, "categoryEntriesStarred", "categoryID", id),
			count, offset, user.EntriesPerPage)).
		Set("showOnlyStarredEntries", true)
	html.OK(w, r, v.Render("category_entries"))
}
