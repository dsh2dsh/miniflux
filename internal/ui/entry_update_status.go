// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	json_parser "encoding/json"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

type entriesStatusSetter func(ctx context.Context, userID int64, status string,
	entryIDs []int64) (int, error)

func (h *handler) updateEntriesStatus(w http.ResponseWriter, r *http.Request) {
	handleEntriesStatus(w, r, h.setEntriesStatus)
}

func (h *handler) setEntriesStatus(ctx context.Context, userID int64,
	status string, entryIDs []int64,
) (int, error) {
	return len(entryIDs), h.store.SetEntriesStatus(ctx, userID, entryIDs, status)
}

func (h *handler) updateEntriesStatusCount(w http.ResponseWriter, r *http.Request) {
	handleEntriesStatus(w, r, h.setEntriesStatusCount)
}

func (h *handler) setEntriesStatusCount(ctx context.Context, userID int64,
	status string, entryIDs []int64,
) (int, error) {
	return h.store.SetEntriesStatusCount(ctx, userID, entryIDs, status)
}

func handleEntriesStatus(w http.ResponseWriter, r *http.Request,
	statusSetterFunc entriesStatusSetter,
) {
	update, err := decodeEntriesStatusUpdate(r)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	count, err := statusSetterFunc(r.Context(), request.UserID(r), update.Status,
		update.EntryIDs)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, count)
}

func decodeEntriesStatusUpdate(r *http.Request,
) (*model.EntriesStatusUpdateRequest, error) {
	updateRequest := new(model.EntriesStatusUpdateRequest)
	err := json_parser.NewDecoder(r.Body).Decode(updateRequest)
	if err != nil {
		return nil, fmt.Errorf(
			"ui: unmarshal entries status update request: %w", err)
	}

	err = validator.ValidateEntriesStatusUpdateRequest(updateRequest)
	if err != nil {
		return nil, err
	}
	return updateRequest, nil
}
