// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) saveCategory(w http.ResponseWriter, r *http.Request) {
	loggedUser, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	categoryForm := form.NewCategoryForm(r)

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("form", categoryForm)
	view.Set("menu", "categories")
	view.Set("user", loggedUser)
	view.Set("countUnread", h.store.CountUnreadEntries(
		r.Context(), loggedUser.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), loggedUser.ID))

	categoryCreationRequest := &model.CategoryCreationRequest{Title: categoryForm.Title}

	validationErr := validator.ValidateCategoryCreation(r.Context(),
		h.store, loggedUser.ID, categoryCreationRequest)
	if validationErr != nil {
		view.Set("errorMessage", validationErr.Translate(loggedUser.Language))
		html.OK(w, r, view.Render("create_category"))
		return
	}

	_, err = h.store.CreateCategory(r.Context(), loggedUser.ID,
		categoryCreationRequest)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	html.Redirect(w, r, route.Path(h.router, "categories"))
}
