// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api // import "miniflux.app/v2/internal/api"

import (
	json_parser "encoding/json"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) createAPIKey(w http.ResponseWriter, r *http.Request,
) (*model.APIKey, error) {
	var createRequest model.APIKeyCreationRequest
	err := json_parser.NewDecoder(r.Body).Decode(&createRequest)
	if err != nil {
		return nil, response.WrapBadRequest(err)
	}

	ctx := r.Context()
	userID := request.UserID(r)
	lerr := validator.ValidateAPIKeyCreation(ctx, h.store, userID, &createRequest)
	if lerr != nil {
		return nil, response.WrapBadRequest(lerr.Error())
	}

	apiKey, err := h.store.CreateAPIKey(ctx, userID, createRequest.Description)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (h *handler) getAPIKeys(w http.ResponseWriter, r *http.Request,
) ([]model.APIKey, error) {
	userID := request.UserID(r)
	apiKeys, err := h.store.APIKeys(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (h *handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) error {
	userID := request.UserID(r)
	id := request.RouteInt64Param(r, "apiKeyID")

	affected, err := h.store.DeleteAPIKey(r.Context(), userID, id)
	if err != nil {
		return err
	} else if !affected {
		return response.ErrNotFound
	}
	return nil
}
