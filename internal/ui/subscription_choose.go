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
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showChooseSubscriptionPage(w http.ResponseWriter,
	r *http.Request,
) {
	f := form.NewSubscriptionForm(r)
	if lerr := f.Validate(); lerr != nil {
		h.showCreateFeedError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}

	userID := request.UserID(r)
	feed, lerr := feedHandler.CreateFeed(r.Context(), h.store, userID,
		&model.FeedCreationRequest{
			CategoryID:                  f.CategoryID,
			FeedURL:                     f.URL,
			Crawler:                     f.Crawler,
			AllowSelfSignedCertificates: f.AllowSelfSignedCertificates,
			UserAgent:                   f.UserAgent,
			Cookie:                      f.Cookie,
			Username:                    f.Username,
			Password:                    f.Password,
			ScraperRules:                f.ScraperRules,
			RewriteRules:                f.RewriteRules,
			UrlRewriteRules:             f.UrlRewriteRules,
			KeepFilterEntryRules:        f.KeepFilterEntryRules,
			BlockFilterEntryRules:       f.BlockFilterEntryRules,
			FetchViaProxy:               f.FetchViaProxy,
			DisableHTTP2:                f.DisableHTTP2,
			ProxyURL:                    f.ProxyURL,
		})
	if lerr != nil {
		h.showCreateFeedError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}
	h.redirect(w, r, "feedEntries", "feedID", feed.ID)
}

func (h *handler) showCreateFeedError(w http.ResponseWriter, r *http.Request,
	renderFunc func(v *View),
) {
	v := h.View(r)

	var categories []model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("categories", categories).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent())
	renderFunc(v)
}
