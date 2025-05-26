// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveAPIKey(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	keyForm := form.NewAPIKeyForm(r)
	keyCreationRequest := model.APIKeyCreationRequest{
		Description: keyForm.Description,
	}

	lerr := validator.ValidateAPIKeyCreation(r.Context(), h.store, v.UserID(),
		&keyCreationRequest)
	if lerr != nil {
		v.Set("menu", "settings").
			Set("form", keyForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_api_key"))
		return
	}

	_, err := h.store.CreateAPIKey(r.Context(), v.UserID(),
		keyCreationRequest.Description)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "apiKeys"))
}
