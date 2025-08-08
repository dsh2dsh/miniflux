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

func (h *handler) showUnreadPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user := v.User()

	query := h.store.NewEntryQueryBuilder(user.ID).
		WithStatus(model.EntryStatusUnread).
		WithGloballyVisible().
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithLimit(user.EntriesPerPage)

	var entries model.Entries
	offset := request.QueryIntParam(r, "offset", 0)
	if offset == 0 {
		v.Go(func(ctx context.Context) (err error) {
			entries, err = query.GetEntries(ctx)
			return
		})
	}

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	if offset != 0 {
		if offset >= v.CountUnread() {
			offset = 0
		}
		e, err := query.WithOffset(offset).GetEntries(r.Context())
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		entries = e
	}

	v.Set("entries", entries).
		Set("menu", "unread").
		Set("pagination", getPagination(route.Path(h.router, "unread"),
			v.CountUnread(), offset, user.EntriesPerPage)).
		Set("updateEntriesStatus", h.router.NamedPath("updateEntriesStatus"))
	html.OK(w, r, v.Render("unread_entries"))
}
