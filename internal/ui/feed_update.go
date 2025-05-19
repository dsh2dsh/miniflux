// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	feedID := request.RouteInt64Param(r, "feedID")
	var feed *model.Feed
	v.Go(func(ctx context.Context) (err error) {
		feed, err = h.store.FeedByID(ctx, v.UserID(), feedID)
		return
	})

	var categories []*model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if feed == nil {
		html.NotFound(w, r)
		return
	}

	feedForm := form.NewFeedForm(r)
	v.Set("menu", "feeds").
		Set("form", feedForm).
		Set("categories", categories).
		Set("feed", feed).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent())

	feedRequest := &model.FeedModificationRequest{
		FeedURL:             model.OptionalString(feedForm.FeedURL),
		SiteURL:             model.OptionalString(feedForm.SiteURL),
		Title:               model.OptionalString(feedForm.Title),
		CommentsURLTemplate: model.OptionalString(feedForm.CommentsURLTemplate),
		Description:         model.OptionalString(feedForm.Description),
		CategoryID:          model.OptionalNumber(feedForm.CategoryID),
		BlocklistRules:      model.OptionalString(feedForm.BlocklistRules),
		KeeplistRules:       model.OptionalString(feedForm.KeeplistRules),
		UrlRewriteRules:     model.OptionalString(feedForm.UrlRewriteRules),
		ProxyURL:            model.OptionalString(feedForm.ProxyURL),
	}

	lerr := validator.ValidateFeedModification(r.Context(), h.store, v.User().ID,
		feedID, feedRequest)
	if lerr != nil {
		v.Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("edit_feed"))
		return
	}

	err := h.store.UpdateFeed(r.Context(), feedForm.Merge(feed))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feedID))
}
