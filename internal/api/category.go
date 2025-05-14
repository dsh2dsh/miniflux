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
	"miniflux.app/v2/internal/validator"
)

func (h *handler) createCategory(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)

	var categoryCreationRequest model.CategoryCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&categoryCreationRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	validationErr := validator.ValidateCategoryCreation(r.Context(),
		h.store, userID, &categoryCreationRequest)
	if validationErr != nil {
		json.BadRequest(w, r, validationErr.Error())
		return
	}

	category, err := h.store.CreateCategory(r.Context(),
		userID, &categoryCreationRequest)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	json.Created(w, r, category)
}

func (h *handler) updateCategory(w http.ResponseWriter, r *http.Request) {
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

	var categoryModificationRequest model.CategoryModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&categoryModificationRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	validationErr := validator.ValidateCategoryModification(r.Context(),
		h.store, userID, category.ID, &categoryModificationRequest)
	if validationErr != nil {
		json.BadRequest(w, r, validationErr.Error())
		return
	}

	categoryModificationRequest.Patch(category)

	if err := h.store.UpdateCategory(r.Context(), category); err != nil {
		json.ServerError(w, r, err)
		return
	}

	json.Created(w, r, category)
}

func (h *handler) markCategoryAsRead(w http.ResponseWriter, r *http.Request) {
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

	err = h.store.MarkCategoryAsRead(r.Context(), userID, categoryID, time.Now())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}

func (h *handler) getCategories(w http.ResponseWriter, r *http.Request) {
	var categories []*model.Category
	var err error
	includeCounts := request.QueryStringParam(r, "counts", "false")

	if includeCounts == "true" {
		categories, err = h.store.CategoriesWithFeedCount(
			r.Context(), request.UserID(r))
	} else {
		categories, err = h.store.Categories(r.Context(), request.UserID(r))
	}

	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, categories)
}

func (h *handler) removeCategory(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	categoryID := request.RouteInt64Param(r, "categoryID")

	if !h.store.CategoryIDExists(r.Context(), userID, categoryID) {
		json.NotFound(w, r)
		return
	}

	err := h.store.RemoveCategory(r.Context(), userID, categoryID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	json.NoContent(w, r)
}

func (h *handler) refreshCategory(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	categoryID := request.RouteInt64Param(r, "categoryID")

	err := h.store.NewBatchBuilder().
		WithErrorLimit(config.Opts.PollingParsingErrorLimit()).
		WithoutDisabledFeeds().
		WithUserID(userID).
		WithCategoryID(categoryID).
		ResetNextCheckAt(r.Context())
	if err != nil {
		json.ServerError(w, r, err)
		return
	}

	slog.Info(
		"Triggered a manual refresh of all feeds for a given category from the API",
		slog.Int64("user_id", userID),
		slog.Int64("category_id", categoryID))

	h.pool.Wakeup()
	json.NoContent(w, r)
}
