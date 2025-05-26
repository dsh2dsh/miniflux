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
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) submitSubscription(w http.ResponseWriter, r *http.Request) {
	f := form.NewSubscriptionForm(r)
	if lerr := f.Validate(); lerr != nil {
		h.showSubmitSubscriptionError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}

	userID := request.UserID(r)
	intg, err := h.store.Integration(r.Context(), userID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	var rssBridgeURL, rssBridgeToken string
	if intg != nil && intg.RSSBridgeEnabled {
		rssBridgeURL = intg.RSSBridgeURL
		rssBridgeToken = intg.Extra.RSSBridgeToken
	}

	requestBuilder := fetcher.NewRequestBuilder().
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(f.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		WithUserAgent(f.UserAgent,
			config.Opts.HTTPClientUserAgent()).
		WithCookie(f.Cookie).
		WithUsernameAndPassword(f.Username,
			f.Password).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		DisableHTTP2(f.DisableHTTP2)

	finder := subscription.NewSubscriptionFinder(requestBuilder)
	subscriptions, lerr := finder.FindSubscriptions(r.Context(),
		f.URL, rssBridgeURL, rssBridgeToken)
	if lerr != nil {
		h.showSubmitSubscriptionError(w, r, func(v *View) {
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}

	n := len(subscriptions)
	switch {
	case n == 0:
		h.showSubmitSubscriptionError(w, r, func(v *View) {
			lerr := locale.NewLocalizedError("error.subscription_not_found")
			v.Set("form", f).
				Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})

	case n == 1 && finder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.CreateFeedFromSubscriptionDiscovery(r.Context(),
			h.store, userID,
			&model.FeedCreationRequestFromSubscriptionDiscovery{
				Content:      finder.FeedResponseInfo().Content,
				ETag:         finder.FeedResponseInfo().ETag,
				LastModified: finder.FeedResponseInfo().LastModified,
				FeedCreationRequest: model.FeedCreationRequest{
					CategoryID:                  f.CategoryID,
					FeedURL:                     subscriptions[0].URL,
					AllowSelfSignedCertificates: f.AllowSelfSignedCertificates,
					Crawler:                     f.Crawler,
					UserAgent:                   f.UserAgent,
					Cookie:                      f.Cookie,
					Username:                    f.Username,
					Password:                    f.Password,
					ScraperRules:                f.ScraperRules,
					RewriteRules:                f.RewriteRules,
					BlocklistRules:              f.BlocklistRules,
					KeeplistRules:               f.KeeplistRules,
					UrlRewriteRules:             f.UrlRewriteRules,
					FetchViaProxy:               f.FetchViaProxy,
					DisableHTTP2:                f.DisableHTTP2,
					ProxyURL:                    f.ProxyURL,
				},
			})
		if lerr != nil {
			h.showSubmitSubscriptionError(w, r, func(v *View) {
				v.Set("form", f).
					Set("errorMessage", lerr.Translate(v.User().Language))
				html.OK(w, r, v.Render("add_subscription"))
			})
			return
		}
		html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feed.ID))

	case n == 1 && !finder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.CreateFeed(r.Context(),
			h.store, userID, &model.FeedCreationRequest{
				CategoryID:                  f.CategoryID,
				FeedURL:                     subscriptions[0].URL,
				Crawler:                     f.Crawler,
				AllowSelfSignedCertificates: f.AllowSelfSignedCertificates,
				UserAgent:                   f.UserAgent,
				Cookie:                      f.Cookie,
				Username:                    f.Username,
				Password:                    f.Password,
				ScraperRules:                f.ScraperRules,
				RewriteRules:                f.RewriteRules,
				BlocklistRules:              f.BlocklistRules,
				KeeplistRules:               f.KeeplistRules,
				UrlRewriteRules:             f.UrlRewriteRules,
				FetchViaProxy:               f.FetchViaProxy,
				DisableHTTP2:                f.DisableHTTP2,
				ProxyURL:                    f.ProxyURL,
			})
		if lerr != nil {
			h.showSubmitSubscriptionError(w, r, func(v *View) {
				v.Set("form", f).
					Set("errorMessage", lerr.Translate(v.User().Language))
				html.OK(w, r, v.Render("add_subscription"))
			})
			return
		}
		html.Redirect(w, r, route.Path(h.router, "feedEntries", "feedID", feed.ID))

	case n > 1:
		h.showSubmitSubscriptionError(w, r, func(v *View) {
			v.Set("subscriptions", subscriptions).
				Set("form", f)
			html.OK(w, r, v.Render("choose_subscription"))
		})
	}
}

func (h *handler) showSubmitSubscriptionError(w http.ResponseWriter,
	r *http.Request, renderFunc func(v *View),
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
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent()).
		Set("hasProxyConfigured", config.Opts.HasHTTPClientProxyURLConfigured())
	renderFunc(v)
}
