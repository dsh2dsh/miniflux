// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	stdjson "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/json"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveEnclosureProgression(w http.ResponseWriter,
	r *http.Request,
) {
	var data model.EnclosureUpdateRequest
	if err := stdjson.NewDecoder(r.Body).Decode(&data); err != nil {
		json.ServerError(w, r, err)
		return
	}

	if err := validator.ValidateEnclosureUpdateRequest(&data); err != nil {
		json.BadRequest(w, r, err)
		return
	}

	enclosure := model.Enclosure{MediaProgression: data.MediaProgression}
	entryID := request.RouteInt64Param(r, "entryID")
	at := request.RouteInt64Param(r, "at")

	ok, err := h.store.UpdateEnclosureAt(r.Context(), request.UserID(r), entryID,
		&enclosure, int(at))
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !ok {
		json.NotFound(w, r)
		return
	}
	json.Created(w, r, map[string]string{"message": "saved"})
}
