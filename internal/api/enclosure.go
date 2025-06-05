// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) getEnclosureByID(w http.ResponseWriter, r *http.Request) {
	id := request.RouteInt64Param(r, "enclosureID")
	enclosure, err := h.store.GetEnclosure(r.Context(), id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if enclosure == nil || enclosure.UserID != request.UserID(r) {
		json.NotFound(w, r)
		return
	}

	enclosure.ProxifyEnclosureURL(h.router)
	json.OK(w, r, enclosure)
}

func (h *handler) updateEnclosureByID(w http.ResponseWriter, r *http.Request) {
	var updateRequest model.EnclosureUpdateRequest
	if err := json_parser.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	err := validator.ValidateEnclosureUpdateRequest(&updateRequest)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	id := request.RouteInt64Param(r, "enclosureID")
	ctx := r.Context()
	enclosure, err := h.store.GetEnclosure(ctx, id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if enclosure == nil || enclosure.UserID != request.UserID(r) {
		json.NotFound(w, r)
		return
	}

	enclosure.MediaProgression = updateRequest.MediaProgression
	if err := h.store.UpdateEnclosure(ctx, enclosure); err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.NoContent(w, r)
}
