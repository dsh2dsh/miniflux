// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/processor"
)

func (h *handler) fetchContent(w http.ResponseWriter, r *http.Request) {
	user := request.User(r)
	entryID := request.RouteInt64Param(r, "entryID")
	g, ctx := errgroup.WithContext(r.Context())

	var entry *model.Entry
	g.Go(func() (err error) {
		entry, err = h.store.NewEntryQueryBuilder(user.ID).
			WithEntryID(entryID).
			WithoutStatus(model.EntryStatusRemoved).
			GetEntry(ctx)
		return
	})

	var feed *model.Feed
	g.Go(func() (err error) {
		feed, err = h.store.FeedByID(ctx, user.ID, entry.FeedID)
		return
	})

	if err := g.Wait(); err != nil {
		json.ServerError(w, r, err)
		return
	} else if entry == nil || feed == nil {
		json.NotFound(w, r)
		return
	}

	ctx = r.Context()
	err := processor.ProcessEntryWebPage(ctx, feed, entry, user)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	err = h.store.UpdateEntryTitleAndContent(ctx, entry)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	content := mediaproxy.RewriteDocumentWithRelativeProxyURL(h.router,
		entry.Content)
	readingTime := locale.NewPrinter(user.Language).Plural(
		"entry.estimated_reading_time", entry.ReadingTime, entry.ReadingTime)

	json.OK(w, r, map[string]string{
		"content":      content,
		"reading_time": readingTime,
	})
}
