// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) createCategory(w http.ResponseWriter, r *http.Request,
) (*model.Category, error) {
	var createRequest model.CategoryCreationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&createRequest); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	userID := request.UserID(r)
	ctx := r.Context()
	lerr := validator.ValidateCategoryCreation(ctx, h.store, userID,
		&createRequest)
	if lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	category, err := h.store.CreateCategory(ctx, userID, &createRequest)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (h *handler) updateCategory(w http.ResponseWriter, r *http.Request,
) (*model.Category, error) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")
	ctx := r.Context()

	category, err := h.store.Category(ctx, userID, id)
	if err != nil {
		return nil, err
	} else if category == nil {
		return nil, response.ErrNotFound
	}

	var modifyRequest model.CategoryModificationRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&modifyRequest); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	lerr := validator.ValidateCategoryModification(ctx, h.store, userID,
		category.ID, &modifyRequest)
	if lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	modifyRequest.Patch(category)
	affected, err := h.store.UpdateCategory(ctx, category)
	if err != nil {
		return nil, err
	} else if !affected {
		return nil, response.ErrNotFound
	}
	return category, nil
}

func (h *handler) markCategoryAsRead(w http.ResponseWriter, r *http.Request,
) error {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	affected, err := h.store.MarkCategoryAsRead(r.Context(), userID, id,
		time.Now())
	if err != nil {
		return err
	} else if !affected {
		return response.ErrNotFound
	}
	return nil
}

func (h *handler) getCategories(w http.ResponseWriter, r *http.Request,
) (categories []model.Category, _ error) {
	var err error
	includeCounts := request.QueryStringParam(r, "counts", "false")

	ctx := r.Context()
	userID := request.UserID(r)
	if includeCounts == "true" {
		categories, err = h.store.CategoriesWithFeedCount(ctx, userID)
	} else {
		categories, err = h.store.Categories(ctx, userID)
	}

	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (h *handler) removeCategory(w http.ResponseWriter, r *http.Request) error {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")

	affected, err := h.store.RemoveCategory(r.Context(), userID, id)
	if err != nil {
		return fmt.Errorf("api: unable remove category id=%v user_id=%v: %w",
			id, userID, err)
	} else if !affected {
		return response.ErrNotFound
	}
	return nil
}

func (h *handler) refreshCategory(w http.ResponseWriter, r *http.Request,
) error {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "categoryID")
	ctx := r.Context()

	err := h.store.NewBatchBuilder().
		WithUserID(userID).
		WithCategoryID(id).
		WithoutDisabledFeeds().
		ResetNextCheckAt(ctx)
	if err != nil {
		return err
	}

	logging.FromContext(ctx).Info(
		"Triggered a manual refresh of all feeds for a given category from the API",
		slog.Int64("user_id", userID),
		slog.Int64("category_id", id))

	h.pool.Wakeup()
	return nil
}
