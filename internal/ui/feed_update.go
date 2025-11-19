// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/ui/form"
	"miniflux.app/v2/internal/validator"
)

func (h *handler) updateFeed(w http.ResponseWriter, r *http.Request) {
	f := form.NewFeedForm(r)
	modify := model.FeedModificationRequest{
		FeedURL:               model.OptionalString(f.FeedURL),
		SiteURL:               model.OptionalString(f.SiteURL),
		Title:                 model.OptionalString(f.Title),
		Description:           model.OptionalString(f.Description),
		CategoryID:            model.OptionalNumber(f.CategoryID),
		UrlRewriteRules:       model.OptionalString(f.UrlRewriteRules),
		BlockFilterEntryRules: model.OptionalString(f.BlockFilterEntryRules),
		KeepFilterEntryRules:  model.OptionalString(f.KeepFilterEntryRules),
		ProxyURL:              model.OptionalString(f.ProxyURL),
		CommentsURLTemplate:   model.OptionalString(f.CommentsURLTemplate),
	}

	ctx := r.Context()
	user := request.User(r)
	feedID := request.RouteInt64Param(r, "feedID")

	lerr := validator.ValidateFeedModification(ctx, h.store, user.ID, feedID,
		&modify)
	if lerr != nil {
		h.showUpdateFeedError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("edit_feed"))
		})
		return
	}

	feed, err := h.store.FeedByID(ctx, user.ID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	err = h.store.UpdateFeed(ctx, f.Merge(feed))
	if err != nil {
		html.ServerError(w, r, err)
		return
	}
	h.redirect(w, r, "feedEntries", "feedID", feedID)
}

func (h *handler) showUpdateFeedError(w http.ResponseWriter, r *http.Request,
	renderFunc func(v *View),
) {
	v := h.View(r)

	feedID := request.RouteInt64Param(r, "feedID")
	var feed *model.Feed
	v.Go(func(ctx context.Context) (err error) {
		feed, err = h.store.FeedByID(ctx, v.UserID(), feedID)
		return err
	})

	var categories []model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	} else if feed == nil {
		html.NotFound(w, r)
		return
	}

	v.Set("menu", "feeds").
		Set("categories", categories).
		Set("feed", feed).
		Set("defaultUserAgent", config.HTTPClientUserAgent())
	renderFunc(v)
}
