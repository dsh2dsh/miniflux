// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/http/route"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) submitSubscription(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)

	var categories []*model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return
	})

	var intg *model.Integration
	v.Go(func(ctx context.Context) (err error) {
		intg, err = h.store.Integration(ctx, v.UserID())
		return
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("categories", categories).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent()).
		Set("hasProxyConfigured", config.Opts.HasHTTPClientProxyURLConfigured())

	subscriptionForm := form.NewSubscriptionForm(r)
	if lerr := subscriptionForm.Validate(); lerr != nil {
		v.Set("form", subscriptionForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("add_subscription"))
		return
	}

	var rssBridgeURL string
	if intg != nil && intg.RSSBridgeEnabled {
		rssBridgeURL = intg.RSSBridgeURL
	}

	requestBuilder := fetcher.NewRequestBuilder().
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(subscriptionForm.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(subscriptionForm.FetchViaProxy).
		WithUserAgent(subscriptionForm.UserAgent,
			config.Opts.HTTPClientUserAgent()).
		WithCookie(subscriptionForm.Cookie).
		WithUsernameAndPassword(subscriptionForm.Username,
			subscriptionForm.Password).
		IgnoreTLSErrors(subscriptionForm.AllowSelfSignedCertificates).
		DisableHTTP2(subscriptionForm.DisableHTTP2)

	subscriptionFinder := subscription.NewSubscriptionFinder(requestBuilder)
	subscriptions, lerr := subscriptionFinder.FindSubscriptions(r.Context(),
		subscriptionForm.URL, rssBridgeURL)
	if lerr != nil {
		v.Set("form", subscriptionForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("add_subscription"))
		return
	}

	n := len(subscriptions)
	switch {
	case n == 0:
		lerr := locale.NewLocalizedError("error.subscription_not_found")
		v.Set("form", subscriptionForm).
			Set("errorMessage", lerr.Translate(v.User().Language))
		html.OK(w, r, v.Render("add_subscription"))

	case n == 1 && subscriptionFinder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.CreateFeedFromSubscriptionDiscovery(r.Context(),
			h.store, v.UserID(),
			&model.FeedCreationRequestFromSubscriptionDiscovery{
				Content:      subscriptionFinder.FeedResponseInfo().Content,
				ETag:         subscriptionFinder.FeedResponseInfo().ETag,
				LastModified: subscriptionFinder.FeedResponseInfo().LastModified,
				FeedCreationRequest: model.FeedCreationRequest{
					CategoryID:                  subscriptionForm.CategoryID,
					FeedURL:                     subscriptions[0].URL,
					AllowSelfSignedCertificates: subscriptionForm.AllowSelfSignedCertificates,
					Crawler:                     subscriptionForm.Crawler,
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
				},
			})
		if lerr != nil {
			v.Set("form", subscriptionForm).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
			return
		}
		html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feed.ID))

	case n == 1 && !subscriptionFinder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.CreateFeed(r.Context(),
			h.store, v.UserID(), &model.FeedCreationRequest{
				CategoryID:                  subscriptionForm.CategoryID,
				FeedURL:                     subscriptions[0].URL,
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

	case n > 1:
		v.Set("subscriptions", subscriptions).
			Set("form", subscriptionForm)
		html.OK(w, r, v.Render("choose_subscription"))
	}
}
