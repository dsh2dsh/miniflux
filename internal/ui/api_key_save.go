// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) saveAPIKey(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	apiKeyForm := form.NewAPIKeyForm(r)
	v.Set("menu", "settings").
		Set("form", apiKeyForm)

	if lerr := apiKeyForm.Validate(); lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_api_key"))
		return
	}

	alreadyExists, err := h.store.APIKeyExists(r.Context(), v.User().ID,
		apiKeyForm.Description)
	if err != nil {
		logging.FromContext(r.Context()).Error("failed API key lookup",
			slog.Int64("user_id", v.User().ID),
			slog.String("description", apiKeyForm.Description),
			slog.Any("error", err))
		html.ServerError(w, r, err)
		return
	}

	if alreadyExists {
		lerr := locale.NewLocalizedError("error.api_key_already_exists")
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("create_api_key"))
		return
	}

	apiKey := model.NewAPIKey(v.User().ID, apiKeyForm.Description)
	if err = h.store.CreateAPIKey(r.Context(), apiKey); err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "apiKeys"))
}
