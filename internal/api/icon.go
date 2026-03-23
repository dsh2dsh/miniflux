// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
)

func (h *handler) getIconByFeedID(w http.ResponseWriter, r *http.Request,
) (*feedIconResponse, error) {
	id := request.RouteInt64Param(r, "feedID")
	icon, err := h.store.IconByFeedID(r.Context(), request.UserID(r), id)
	if err != nil {
		return nil, err
	} else if icon == nil {
		return nil, response.ErrNotFound
	}

	return &feedIconResponse{
		ID:       icon.ID,
		MimeType: icon.MimeType,
		Data:     icon.DataURL(),
	}, nil
}

func (h *handler) getIconByIconID(w http.ResponseWriter, r *http.Request,
) (*feedIconResponse, error) {
	id := request.RouteInt64Param(r, "iconID")
	icon, err := h.store.IconByID(r.Context(), id)
	if err != nil {
		return nil, err
	} else if icon == nil {
		return nil, response.ErrNotFound
	}

	return &feedIconResponse{
		ID:       icon.ID,
		MimeType: icon.MimeType,
		Data:     icon.DataURL(),
	}, nil
}
