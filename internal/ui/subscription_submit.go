// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	feedHandler "miniflux.app/v2/internal/reader/handler"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/reader/subscription"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) submitSubscription(w http.ResponseWriter, r *http.Request) {
	f := form.NewSubscriptionForm(r)
	if lerr := f.Validate(); lerr != nil {
		h.showSubscriptionError(w, r, f, nil, func(v *View) {
			v.Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}

	user := request.User(r)
	requestBuilder := fetcher.NewRequestBuilder().
		WithCustomFeedProxyURL(f.ProxyURL).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		WithUserAgent(f.UserAgent, config.HTTPClientUserAgent()).
		WithCookie(f.Cookie).
		WithUsernameAndPassword(f.Username, f.Password).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		DisableHTTP2(f.DisableHTTP2)

	ctx := r.Context()
	finder := subscription.NewSubscriptionFinder(requestBuilder)
	subscriptions, lerr := finder.FindSubscriptions(ctx, f.URL,
		user.Integration().RSSBridgeURLIfEnabled(),
		user.Integration().RSSBridgeTokenIfEnabled())
	if lerr != nil {
		h.showSubscriptionError(w, r, f, lerr, func(v *View) {
			html.OK(w, r, v.Render("add_subscription"))
		})
		return
	}

	n := len(subscriptions)
	switch {
	case n == 0:
		h.showSubscriptionError(w, r, f, nil, func(v *View) {
			lerr := locale.NewLocalizedError("error.subscription_not_found")
			v.Set("errorMessage", lerr.Translate(v.User().Language))
			html.OK(w, r, v.Render("add_subscription"))
		})

	case n == 1 && finder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.New(h.store, user.ID, h.tpl).FromDiscovery(
			ctx,
			&model.FeedCreationRequestFromSubscriptionDiscovery{
				Content:      finder.FeedResponseInfo().Content,
				ETag:         finder.FeedResponseInfo().ETag,
				LastModified: finder.FeedResponseInfo().LastModified,
				FeedCreationRequest: model.FeedCreationRequest{
					CategoryID:                  f.CategoryID,
					FeedURL:                     subscriptions[0].URL,
					AllowSelfSignedCertificates: f.AllowSelfSignedCertificates,
					Crawler:                     f.Crawler,
					IgnoreEntryUpdates:          f.IgnoreEntryUpdates,
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
				},
			})
		if lerr != nil {
			h.showSubscriptionError(w, r, f, lerr, func(v *View) {
				html.OK(w, r, v.Render("add_subscription"))
			})
			return
		}
		h.redirect(w, r, "feedEntries", "feedID", feed.ID)

	case n == 1 && !finder.IsFeedAlreadyDownloaded():
		feed, lerr := feedHandler.New(h.store, user.ID, h.tpl).FromRequest(
			ctx,
			&model.FeedCreationRequest{
				CategoryID:                  f.CategoryID,
				FeedURL:                     subscriptions[0].URL,
				Crawler:                     f.Crawler,
				IgnoreEntryUpdates:          f.IgnoreEntryUpdates,
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
			h.showSubscriptionError(w, r, f, lerr, func(v *View) {
				html.OK(w, r, v.Render("add_subscription"))
			})
			return
		}
		h.redirect(w, r, "feedEntries", "feedID", feed.ID)

	case n > 1:
		h.showSubscriptionError(w, r, f, nil, func(v *View) {
			v.Set("subscriptions", subscriptions)
			html.OK(w, r, v.Render("choose_subscription"))
		})
	}
}

func (h *handler) showSubscriptionError(w http.ResponseWriter, r *http.Request,
	f *form.SubscriptionForm, lerr *locale.LocalizedErrorWrapper,
	renderFunc func(v *View),
) {
	v := h.View(r)

	var categories []model.Category
	v.Go(func(ctx context.Context) (err error) {
		categories, err = h.store.Categories(ctx, v.UserID())
		return err
	})

	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	v.Set("menu", "feeds").
		Set("categories", categories).
		Set("defaultUserAgent", config.HTTPClientUserAgent()).
		Set("form", f).
		Set("hasProxyConfigured", config.HasHTTPClientProxyURLConfigured())

	if lerr == nil {
		renderFunc(v)
		return

	}

	v.Set("errorMessage", lerr.Translate(v.User().Language))
	v.Set("badStatusContent",
		template.HTML(h.badStatusContent(r.Context(), f.URL, lerr)))
	renderFunc(v)
}

func (h *handler) badStatusContent(ctx context.Context, urlString string,
	err error,
) string {
	badStatusErr, ok := errors.AsType[*fetcher.ErrBadStatus](err)
	if !ok || len(badStatusErr.Body) == 0 {
		return ""
	}

	u, err := url.Parse(urlString)
	if err != nil {
		logging.FromContext(ctx).Error("unable parse content URL",
			slog.String("url", urlString), slog.Any("error", err))
		return ""
	}

	return sanitizer.SanitizeContent(string(badStatusErr.Body), u,
		sanitizer.WithRewriteURL(mediaproxy.New(h.router).RewriteURL))
}
