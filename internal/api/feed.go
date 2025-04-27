// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) createFeed(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)

	var feedCreationRequest model.FeedCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&feedCreationRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	// Make the feed category optional for clients who don't support categories.
	if feedCreationRequest.CategoryID == 0 {
		category, err := h.store.FirstCategory(r.Context(), userID)
		if err != nil {
			json.ServerError(w, r, err)
			return
		}
		feedCreationRequest.CategoryID = category.ID
	}

	validationErr := validator.ValidateFeedCreation(r.Context(),
		h.store, userID, &feedCreationRequest)
	if validationErr != nil {
		json.BadRequest(w, r, validationErr.Error())
		return
	}

	feed, localizedError := feedHandler.CreateFeed(r.Context(),
		h.store, userID, &feedCreationRequest)
	if localizedError != nil {
		json.ServerError(w, r, localizedError.Error())
		return
	}

	json.Created(w, r, &feedCreationResponse{FeedID: feed.ID})
}

func (h *handler) refreshFeed(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	if !h.store.FeedExists(r.Context(), userID, feedID) {
		json.NotFound(w, r)
		return
	}

	localizedError := feedHandler.RefreshFeed(r.Context(), h.store, userID,
		feedID, false)
	if localizedError != nil {
		json.ServerError(w, r, localizedError.Error())
		return
	}

	json.NoContent(w, r)
}

func (h *handler) refreshAllFeeds(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)

	batchBuilder := h.store.NewBatchBuilder()
	batchBuilder.WithErrorLimit(config.Opts.PollingParsingErrorLimit())
	batchBuilder.WithoutDisabledFeeds()
	batchBuilder.WithNextCheckExpired()
	batchBuilder.WithUserID(userID)

	jobs, err := batchBuilder.FetchJobs(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	slog.Info(
		"Triggered a manual refresh of all feeds from the API",
		slog.Int64("user_id", userID),
		slog.Int("nb_jobs", len(jobs)),
	)

	go h.pool.Push(jobs)

	json.NoContent(w, r)
}

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	var feedModificationRequest model.FeedModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&feedModificationRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	userID := request.UserID(r)
	feedID := request.RouteInt64Param(r, "feedID")

	originalFeed, err := h.store.FeedByID(r.Context(), userID, feedID)
	if err != nil {
		json.NotFound(w, r)
		return
	}

	if originalFeed == nil {
		json.NotFound(w, r)
		return
	}

	validationErr := validator.ValidateFeedModification(r.Context(),
		h.store, userID, originalFeed.ID, &feedModificationRequest)
	if validationErr != nil {
		json.BadRequest(w, r, validationErr.Error())
		return
	}

	feedModificationRequest.Patch(originalFeed)
	originalFeed.ResetErrorCounter()
	if err := h.store.UpdateFeed(r.Context(), originalFeed); err != nil {
		json.ServerError(w, r, err)
		return
	}

	originalFeed, err = h.store.FeedByID(r.Context(), userID, feedID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	json.Created(w, r, originalFeed)
}

func (h *handler) markFeedAsRead(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	feed, err := h.store.FeedByID(r.Context(), userID, feedID)
	if err != nil {
		json.NotFound(w, r)
		return
	}

	if feed == nil {
		json.NotFound(w, r)
		return
	}

	err = h.store.MarkFeedAsRead(r.Context(), userID, feedID, time.Now())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) getCategoryFeeds(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	categoryID := request.RouteInt64Param(r, "categoryID")

	category, err := h.store.Category(r.Context(), userID, categoryID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	if category == nil {
		json.NotFound(w, r)
		return
	}

	feeds, err := h.store.FeedsByCategoryWithCounters(
		r.Context(), userID, categoryID)
	if err != nil {
		json.ServerError(w, r, err)
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
	feedID := request.RouteInt64Param(r, "feedID")
	feed, err := h.store.FeedByID(r.Context(), request.UserID(r), feedID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	if feed == nil {
		json.NotFound(w, r)
		return
	}

	json.OK(w, r, feed)
}

func (h *handler) removeFeed(w http.ResponseWriter, r *http.Request) {
	feedID := request.RouteInt64Param(r, "feedID")
	userID := request.UserID(r)

	if !h.store.FeedExists(r.Context(), userID, feedID) {
		json.NotFound(w, r)
		return
	}

	if err := h.store.RemoveFeed(r.Context(), userID, feedID); err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}
