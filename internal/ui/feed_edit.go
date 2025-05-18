// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"net/http"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/ui/form"
)

func (h *handler) showEditFeedPage(w http.ResponseWriter, r *http.Request) {
	v := h.View(r)
	if err := v.Wait(); err != nil {
		html.ServerError(w, r, err)
		return
	}

	feedID := request.RouteInt64Param(r, "feedID")
	feed, err := h.store.FeedByID(r.Context(), v.User().ID, feedID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	} else if feed == nil {
		html.NotFound(w, r)
		return
	}

	categories, err := h.store.Categories(r.Context(), v.User().ID)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	feedForm := form.FeedForm{
		SiteURL:                     feed.SiteURL,
		FeedURL:                     feed.FeedURL,
		Title:                       feed.Title,
		CommentsURLTemplate:         feed.Extra.CommentsURLTemplate,
		Description:                 feed.Description,
		ScraperRules:                feed.ScraperRules,
		RewriteRules:                feed.RewriteRules,
		BlocklistRules:              feed.BlocklistRules,
		KeeplistRules:               feed.KeeplistRules,
		UrlRewriteRules:             feed.UrlRewriteRules,
		Crawler:                     feed.Crawler,
		UserAgent:                   feed.UserAgent,
		Cookie:                      feed.Cookie,
		CategoryID:                  feed.Category.ID,
		Username:                    feed.Username,
		Password:                    feed.Password,
		IgnoreHTTPCache:             feed.IgnoreHTTPCache,
		AllowSelfSignedCertificates: feed.AllowSelfSignedCertificates,
		FetchViaProxy:               feed.FetchViaProxy,
		Disabled:                    feed.Disabled,
		NoMediaPlayer:               feed.NoMediaPlayer,
		HideGlobally:                feed.HideGlobally,
		CategoryHidden:              feed.Category.HideGlobally,
		AppriseServiceURLs:          feed.AppriseServiceURLs,
		WebhookURL:                  feed.WebhookURL,
		DisableHTTP2:                feed.DisableHTTP2,
		NtfyEnabled:                 feed.NtfyEnabled,
		NtfyPriority:                feed.NtfyPriority,
		NtfyTopic:                   feed.NtfyTopic,
		PushoverEnabled:             feed.PushoverEnabled,
		PushoverPriority:            feed.PushoverPriority,
		ProxyURL:                    feed.ProxyURL,
	}

	v.Set("menu", "feeds").
		Set("form", feedForm).
		Set("categories", categories).
		Set("feed", feed).
		Set("defaultUserAgent", config.Opts.HTTPClientUserAgent()).
		Set("hasProxyConfigured", config.Opts.HasHTTPClientProxyURLConfigured())
	html.OK(w, r, v.Render("edit_feed"))
}
