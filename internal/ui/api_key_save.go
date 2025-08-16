// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveAPIKey(w http.ResponseWriter, r *http.Request) {
	f := form.NewAPIKeyForm(r)
	createRequest := model.APIKeyCreationRequest{Description: f.Description}

	userID := request.UserID(r)
	lerr := validator.ValidateAPIKeyCreation(r.Context(), h.store, userID,
		&createRequest)
	if lerr == nil {
		_, err := h.store.CreateAPIKey(r.Context(), userID,
			createRequest.Description)
		if err != nil {
			html.ServerError(w, r, err)
			return
		}
		h.redirect(w, r, "apiKeys")
		return
	}

	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "settings").
		Set("form", f).
		Set("errorMessage", lerr.Translate(v.User().Language))
	html.OK(w, r, v.Render("create_api_key"))
}
