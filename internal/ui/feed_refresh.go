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
	"miniflux.app/v2/internal/locale"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/ui/session"
)

func (h *handler) refreshFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)
	feedID := request.RouteInt64Param(r, "feedID")
	force := request.QueryBoolParam(r, "forceRefresh", false)

	_, err := feedHandler.RefreshFeed(ctx, h.store, userID, feedID, force)
	if err != nil {
		slog.Warn("Unable to refresh feed",
			slog.Int64("user_id", request.UserID(r)),
			slog.Int64("feed_id", feedID),
			slog.Bool("force_refresh", force),
			slog.Any("error", err))
		session.New(h.store, r).NewFlashErrorMessage(err.Error()).Commit(ctx)
	}
	h.redirect(w, r, "feedEntries", "feedID", feedID)
}

func (h *handler) refreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	printer := locale.NewPrinter(request.UserLanguage(r))

	sess := session.New(h.store, r)
	defer sess.Commit(r.Context())

	// Avoid accidental and excessive refreshes.
	sinceLastRefresh := time.Now().UTC().Unix() - request.LastForceRefresh(r)
	refreshInterval := int64(config.Opts.ForceRefreshInterval()) * 60
	if sinceLastRefresh < refreshInterval {
		time := config.Opts.ForceRefreshInterval()
		sess.NewFlashErrorMessage(printer.Plural(
			"alert.too_many_feeds_refresh", time, time))
		h.redirect(w, r, "feeds")
		return
	}

	// We allow the end-user to force refresh all its feeds without taking into
	// consideration the number of errors.
	err := h.store.NewBatchBuilder().
		WithUserID(userID).
		WithoutDisabledFeeds().
		ResetNextCheckAt(r.Context())
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	slog.Info(
		"Triggered a manual refresh of all feeds from the web ui",
		slog.Int64("user_id", userID))

	sess.SetLastForceRefresh().
		NewFlashMessage(printer.Print("alert.background_feed_refresh"))
	h.pool.Wakeup()
	h.redirect(w, r, "feeds")
}
