// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) createFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	var createRequest model.FeedCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	// Make the feed category optional for clients who don't support categories.
	if createRequest.CategoryID == 0 {
		category, err := h.store.FirstCategory(ctx, userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
		createRequest.CategoryID = category.ID
	}

	lerr := validator.ValidateFeedCreation(ctx, h.store, userID, &createRequest)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	feed, localizedError := feedHandler.CreateFeed(ctx,
		h.store, userID, &createRequest)
	if localizedError != nil {
		json.ServerError(w, r, localizedError)
		return
	}
	json.Created(w, r, &feedCreationResponse{FeedID: feed.ID})
}

func (h *handler) refreshFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	if !h.store.FeedExists(ctx, userID, id) {
		json.NotFound(w, r)
		return
	}

	err := feedHandler.RefreshFeed(ctx, h.store, userID, id, false)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) refreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := request.UserID(r)

	err := h.store.NewBatchBuilder().
		WithUserID(userID).
		WithoutDisabledFeeds().
		WithErrorLimit(config.Opts.PollingParsingErrorLimit()).
		ResetNextCheckAt(ctx)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	logging.FromContext(ctx).Info(
		"Triggered a manual refresh of all feeds from the API",
		slog.Int64("user_id", userID))

	h.pool.Wakeup()
	json.NoContent(w, r)
}

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	var modifyRequest model.FeedModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&modifyRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	ctx := r.Context()
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "feedID")

	feed, err := h.store.FeedByID(ctx, userID, id)
	if err != nil {
		json.NotFound(w, r)
		return
	} else if feed == nil {
		json.NotFound(w, r)
		return
	}

	lerr := validator.ValidateFeedModification(ctx, h.store, userID,
		feed.ID, &modifyRequest)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	modifyRequest.Patch(feed)
	feed.ResetErrorCounter()
	if err := h.store.UpdateFeed(ctx, feed); err != nil {
		json.ServerError(w, r, err)
		return
	}

	feed, err = h.store.FeedByID(ctx, userID, id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Created(w, r, feed)
}

func (h *handler) markFeedAsRead(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	affected, err := h.store.MarkFeedAsRead(r.Context(), userID, id, time.Now())
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !affected {
		json.NotFound(w, r)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) getCategoryFeeds(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	g, ctx := errgroup.WithContext(r.Context())
	var category *model.Category
	g.Go(func() (err error) {
		category, err = h.store.Category(ctx, userID, id)
		return
	})

	var feeds model.Feeds
	g.Go(func() (err error) {
		feeds, err = h.store.FeedsByCategoryWithCounters(ctx, userID, id)
		return
	})

	if err := g.Wait(); err != nil {
		json.ServerError(w, r, err)
		return
	} else if category == nil {
		json.NotFound(w, r)
		return
	}
	json.OK(w, r, feeds)
}

func (h *handler) getFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.store.Feeds(r.Context(), request.UserID(r))
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, feeds)
}

func (h *handler) fetchCounters(w http.ResponseWriter, r *http.Request) {
	counters, err := h.store.FetchCounters(r.Context(), request.UserID(r))
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, counters)
}

func (h *handler) getFeed(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "feedID")
	feed, err := h.store.FeedByID(r.Context(), request.UserID(r), id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if feed == nil {
		json.NotFound(w, r)
		return
	}
	json.OK(w, r, feed)
}

func (h *handler) removeFeed(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	affected, err := h.store.RemoveFeed(r.Context(), userID, id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !affected {
		json.NotFound(w, r)
		return
	}
	json.NoContent(w, r)
}
