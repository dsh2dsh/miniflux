// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/ui/session"
	"miniflux.app/v2/internal/ui/view"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	loggedUser, err := h.store.UserByID(r.Context(), request.UserID(r))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	feedID := request.RouteInt64Param(r, "feedID")
	feed, err := h.store.FeedByID(r.Context(), loggedUser.ID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	if feed == nil {
		html.NotFound(w, r)
		return
	}

	categories, err := h.store.Categories(r.Context(), loggedUser.ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	feedForm := form.NewFeedForm(r)

	sess := session.New(h.store, request.SessionID(r))
	view := view.New(h.tpl, r, sess)
	view.Set("form", feedForm)
	view.Set("categories", categories)
	view.Set("feed", feed)
	view.Set("menu", "feeds")
	view.Set("user", loggedUser)
	view.Set("countUnread", h.store.CountUnreadEntries(
		r.Context(), loggedUser.ID))
	view.Set("countErrorFeeds", h.store.CountUserFeedsWithErrors(
		r.Context(), loggedUser.ID))
	view.Set("defaultUserAgent", config.Opts.HTTPClientUserAgent())

	feedModificationRequest := &model.FeedModificationRequest{
		FeedURL:         model.OptionalString(feedForm.FeedURL),
		SiteURL:         model.OptionalString(feedForm.SiteURL),
		Title:           model.OptionalString(feedForm.Title),
		Description:     model.OptionalString(feedForm.Description),
		CategoryID:      model.OptionalNumber(feedForm.CategoryID),
		BlocklistRules:  model.OptionalString(feedForm.BlocklistRules),
		KeeplistRules:   model.OptionalString(feedForm.KeeplistRules),
		UrlRewriteRules: model.OptionalString(feedForm.UrlRewriteRules),
		ProxyURL:        model.OptionalString(feedForm.ProxyURL),
	}

	validationErr := validator.ValidateFeedModification(r.Context(),
		h.store, loggedUser.ID, feed.ID, feedModificationRequest)
	if validationErr != nil {
		view.Set("errorMessage", validationErr.Translate(loggedUser.Language))
		html.OK(w, r, view.Render("edit_feed"))
		return
	}

	err = h.store.UpdateFeed(r.Context(), feedForm.Merge(feed))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feed.ID))
}
