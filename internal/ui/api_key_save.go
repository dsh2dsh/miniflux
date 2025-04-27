// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
)

func (h *handler) saveAPIKey(w http.ResponseWriter, r *http.Request) {
	user, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	apiKeyForm := form.NewAPIKeyForm(r)

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("form", apiKeyForm)
	view.Set("menu", "settings")
	view.Set("user", user)
	view.Set("countUnread", h.store.CountUnreadEntries(r.Context(), user.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), user.ID))

	if validationErr := apiKeyForm.Validate(); validationErr != nil {
		view.Set("errorMessage", validationErr.Translate(user.Language))
		html.OK(w, r, view.Render("create_api_key"))
		return
	}

	alreadyExists, err := h.store.APIKeyExists(r.Context(), user.ID,
		apiKeyForm.Description)
	if err != nil {
		logging.FromContext(r.Context()).Error("failed API key lookup",
			slog.Int64("user_id", user.ID),
			slog.String("description", apiKeyForm.Description),
			slog.Any("error", err))
		html.ServerError(w, r, err)
		return
	}

	if alreadyExists {
		view.Set("errorMessage", locale.NewLocalizedError("error.api_key_already_exists").Translate(user.Language))
		html.OK(w, r, view.Render("create_api_key"))
		return
	}

	apiKey := model.NewAPIKey(user.ID, apiKeyForm.Description)
	if err = h.store.CreateAPIKey(r.Context(), apiKey); err != nil {
		html.ServerError(w, r, err)
		return
	}

	html.Redirect(w, r, route.Path(h.router, "apiKeys"))
}
