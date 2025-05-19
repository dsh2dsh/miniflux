// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
)

func (h *handler) showUnreadPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r).WithSaveEntry()
	user, err := v.WaitUser()
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	query := h.store.NewEntryQueryBuilder(user.ID).
		WithStatus(model.EntryStatusUnread).
		WithSorting(user.EntryOrder, user.EntryDirection).
		WithSorting("id", user.EntryDirection).
		WithLimit(user.EntriesPerPage).
		WithGloballyVisible()

	var startTime time.Time
	var fetchEntriesElapsed time.Duration
	var entries model.Entries

	offset := request.QueryIntParam(r, "offset", 0)
	if offset == 0 {
		startTime = time.Now()
		v.Go(func(ctx context.Context) (err error) {
			entries, err = query.GetEntries(ctx)
			fetchEntriesElapsed = time.Since(startTime)
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
		startTime = time.Now()
		e, err := query.WithOffset(offset).GetEntries(r.Context())
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		entries = e
		fetchEntriesElapsed = time.Since(startTime)
	}

	b := v.Set("menu", "unread").
		Set("entries", entries).
		Set("pagination", getPagination(route.Path(h.router, "unread"),
			v.CountUnread(), offset, user.EntriesPerPage)).
		Render("unread_entries")

	if config.Opts.HasServerTimingHeader() {
		w.Header().Set("Server-Timing", fmt.Sprintf(
			"pre_processing;dur=%d,sql_count_unread_entries;dur=%d,sql_fetch_unread_entries;dur=%d,template_rendering;dur=%d",
			v.PreProcessingElapsed().Milliseconds(),
			v.CountUnreadElapsed().Milliseconds(),
			fetchEntriesElapsed.Milliseconds(),
			v.RenderingElapsed().Milliseconds(),
		))
	}
	html.OK(w, r, b)
}
