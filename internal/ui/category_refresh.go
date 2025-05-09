// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) refreshCategoryEntriesPage(w http.ResponseWriter, r *http.Request) {
	categoryID := h.refreshCategory(w, r)
	html.Redirect(w, r, route.Path(h.router, "categoryEntries", "categoryID", categoryID))
}

func (h *handler) refreshCategoryFeedsPage(w http.ResponseWriter, r *http.Request) {
	categoryID := h.refreshCategory(w, r)
	html.Redirect(w, r, route.Path(h.router, "categoryFeeds", "categoryID", categoryID))
}

func (h *handler) refreshCategory(w http.ResponseWriter, r *http.Request) int64 {
	userID := request.UserID(r)
	categoryID := request.RouteInt64Param(r, "categoryID")
	printer := locale.NewPrinter(request.UserLanguage(r))
	sess := session.New(h.store, request.SessionID(r))

	// Avoid accidental and excessive refreshes.
	if time.Now().UTC().Unix()-request.LastForceRefresh(r) < int64(config.Opts.ForceRefreshInterval())*60 {
		time := config.Opts.ForceRefreshInterval()
		sess.NewFlashErrorMessage(r.Context(),
			printer.Plural("alert.too_many_feeds_refresh", time, time))
	} else {
		// We allow the end-user to force refresh all its feeds in this category
		// without taking into consideration the number of errors.
		batchBuilder := h.store.NewBatchBuilder()
		batchBuilder.WithoutDisabledFeeds()
		batchBuilder.WithUserID(userID)
		batchBuilder.WithCategoryID(categoryID)

		jobs, err := batchBuilder.FetchJobs(r.Context())
		if err != nil {
			html.ServerError(w, r, err)
			return 0
		}

		slog.Info(
			"Triggered a manual refresh of all feeds for a given category from the web ui",
			slog.Int64("user_id", userID),
			slog.Int64("category_id", categoryID),
			slog.Int("nb_jobs", len(jobs)),
		)

		h.pool.Push(r.Context(), jobs)
		sess.SetLastForceRefresh(r.Context())
		sess.NewFlashMessage(r.Context(),
			printer.Print("alert.background_feed_refresh"))
	}

	return categoryID
}
