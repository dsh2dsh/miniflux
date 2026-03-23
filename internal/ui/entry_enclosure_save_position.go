// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	stdjson "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveEnclosureProgression(w http.ResponseWriter,
	r *http.Request,
) (map[string]string, error) {
	var data model.EnclosureUpdateRequest
	if err := stdjson.NewDecoder(r.Body).Decode(&data); err != nil {
		return nil, response.WrapServerError(err)
	}

	if err := validator.ValidateEnclosureUpdateRequest(&data); err != nil {
		return nil, response.WrapBadRequest(err)
	}

	enclosure := model.Enclosure{MediaProgression: data.MediaProgression}
	entryID := request.RouteInt64Param(r, "entryID")
	at := request.RouteInt64Param(r, "at")

	ok, err := h.store.UpdateEnclosureAt(r.Context(), request.UserID(r), entryID,
		&enclosure, int(at))
	if err != nil {
		return nil, response.WrapServerError(err)
	} else if !ok {
		return nil, response.ErrNotFound
	}
	return map[string]string{"message": "saved"}, nil
}
