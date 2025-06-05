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

func (h *handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var createRequest model.APIKeyCreationRequest
	err := json_parser.NewDecoder(r.Body).Decode(&createRequest)
	if err != nil {
		json.BadRequest(w, r, err)
		return
	}

	ctx := r.Context()
	userID := request.UserID(r)
	lerr := validator.ValidateAPIKeyCreation(ctx, h.store, userID, &createRequest)
	if lerr != nil {
		json.BadRequest(w, r, lerr.Error())
		return
	}

	apiKey, err := h.store.CreateAPIKey(ctx, userID, createRequest.Description)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.Created(w, r, apiKey)
}

func (h *handler) getAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	apiKeys, err := h.store.APIKeys(r.Context(), userID)
	if err != nil {
		json.ServerError(w, r, err)
		return
	}
	json.OK(w, r, apiKeys)
}

func (h *handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "apiKeyID")

	affected, err := h.store.DeleteAPIKey(r.Context(), userID, id)
	if err != nil {
		json.ServerError(w, r, err)
		return
	} else if !affected {
		json.NotFound(w, r)
		return
	}
	json.NoContent(w, r)
}
