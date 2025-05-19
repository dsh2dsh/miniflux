// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/model"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showChooseSubscriptionPage(w http.ResponseWriter,
	r *http.Request,
) {
	v := h.View(r)

	var categories []*model.Category
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

	subscriptionForm := form.NewSubscriptionForm(r)
	if lerr := subscriptionForm.Validate(); lerr != nil {
		v.Set("form", subscriptionForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("add_subscription"))
		return
	}

	feed, lerr := feedHandler.CreateFeed(r.Context(),
		h.store, v.User().ID, &model.FeedCreationRequest{
			CategoryID:                  subscriptionForm.CategoryID,
			FeedURL:                     subscriptionForm.URL,
			Crawler:                     subscriptionForm.Crawler,
			AllowSelfSignedCertificates: subscriptionForm.AllowSelfSignedCertificates,
			UserAgent:                   subscriptionForm.UserAgent,
			Cookie:                      subscriptionForm.Cookie,
			Username:                    subscriptionForm.Username,
			Password:                    subscriptionForm.Password,
			ScraperRules:                subscriptionForm.ScraperRules,
			RewriteRules:                subscriptionForm.RewriteRules,
			BlocklistRules:              subscriptionForm.BlocklistRules,
			KeeplistRules:               subscriptionForm.KeeplistRules,
			UrlRewriteRules:             subscriptionForm.UrlRewriteRules,
			FetchViaProxy:               subscriptionForm.FetchViaProxy,
			DisableHTTP2:                subscriptionForm.DisableHTTP2,
			ProxyURL:                    subscriptionForm.ProxyURL,
		})
	if lerr != nil {
		v.Set("form", subscriptionForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("add_subscription"))
		return
	}
	html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feed.ID))
}
