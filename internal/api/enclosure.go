// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	stdjson "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateEnclosureAt(w http.ResponseWriter, r *http.Request,
) error {
	var data model.EnclosureUpdateRequest
	if err := stdjson.NewDecoder(r.Body).Decode(&data); err != nil {
		return response.WrapBadRequest(err)
	}

	if err := validator.ValidateEnclosureUpdateRequest(&data); err != nil {
		return response.WrapBadRequest(err)
	}

	enclosure := model.Enclosure{MediaProgression: data.MediaProgression}
	entryID := request.RouteInt64Param(r, "entryID")
	at := request.RouteInt64Param(r, "at")

	ok, err := h.store.UpdateEnclosureAt(r.Context(), request.UserID(r), entryID,
		&enclosure, int(at))
	if err != nil {
		return err
	} else if !ok {
		return response.ErrNotFound
	}
	return nil
}
