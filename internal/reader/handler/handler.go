// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler // import "miniflux.app/v2/internal/reader/handler"

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/icon"
	"miniflux.app/v2/internal/reader/parser"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/storage"
)

var (
	ErrCategoryNotFound = errors.New("fetcher: category not found")
	ErrFeedNotFound     = errors.New("fetcher: feed not found")
	ErrDuplicatedFeed   = errors.New("fetcher: duplicated feed")
)

func CreateFeedFromSubscriptionDiscovery(ctx context.Context,
	store *storage.Storage, userID int64,
	r *model.FeedCreationRequestFromSubscriptionDiscovery,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID),
		slog.String("feed_url", r.FeedURL),
		slog.String("proxy_url", r.ProxyURL))
	log.Debug("Begin feed creation process from subscription discovery")

	if !store.CategoryIDExists(ctx, userID, r.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound,
			"error.category_not_found")
	}

	if store.FeedURLExists(ctx, userID, r.FeedURL) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
	}

	return createFeed(logging.WithLogger(ctx, log), store, userID,
		&r.FeedCreationRequest, r.FeedURL, r.ETag, r.LastModified, r.Content)
}

// CreateFeed fetch, parse and store a new feed.
func CreateFeed(ctx context.Context, store *storage.Storage, userID int64,
	r *model.FeedCreationRequest,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID),
		slog.String("feed_url", r.FeedURL),
		slog.String("proxy_url", r.ProxyURL))
	log.Debug("Begin feed creation process")

	if !store.CategoryIDExists(ctx, userID, r.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound,
			"error.category_not_found")
	}

	requestBuilder := fetcher.NewRequestBuilder().
		WithUsernameAndPassword(r.Username, r.Password).
		WithUserAgent(r.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(r.Cookie).
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(r.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(r.FetchViaProxy).
		IgnoreTLSErrors(r.AllowSelfSignedCertificates).
		DisableHTTP2(r.DisableHTTP2)

	resp, err := fetcher.NewResponseSemaphore(ctx, requestBuilder, r.FeedURL)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_feed",
			err)
	}
	defer resp.Close()

	if lerr := resp.LocalizedError(); lerr != nil {
		log.Warn("Unable to fetch feed", slog.Any("error", lerr))
		return nil, lerr
	}

	if store.FeedURLExists(ctx, userID, resp.EffectiveURL()) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
	}

	body, lerr := resp.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if lerr != nil {
		log.Warn("Unable to fetch feed", slog.Any("error", lerr))
		return nil, lerr
	}
	resp.Close()

	return createFeed(logging.WithLogger(ctx, log), store, userID, r,
		resp.EffectiveURL(), resp.ETag(), resp.LastModified(), body)
}

func createFeed(ctx context.Context, store *storage.Storage, userID int64,
	r *model.FeedCreationRequest, url, etag, lastModified string, body []byte,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	feed, err := parser.ParseFeed(url, bytes.NewReader(body))
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_feed", err)
	}

	feed.UserID = userID
	feed.UserAgent = r.UserAgent
	feed.Cookie = r.Cookie
	feed.Username = r.Username
	feed.Password = r.Password
	feed.Crawler = r.Crawler
	feed.Disabled = r.Disabled
	feed.IgnoreHTTPCache = r.IgnoreHTTPCache
	feed.AllowSelfSignedCertificates = r.AllowSelfSignedCertificates
	feed.DisableHTTP2 = r.DisableHTTP2
	feed.FetchViaProxy = r.FetchViaProxy
	feed.ScraperRules = r.ScraperRules
	feed.RewriteRules = r.RewriteRules
	feed.BlocklistRules = r.BlocklistRules
	feed.KeeplistRules = r.KeeplistRules
	feed.UrlRewriteRules = r.UrlRewriteRules
	feed.HideGlobally = r.HideGlobally
	feed.EtagHeader = etag
	feed.LastModifiedHeader = lastModified
	feed.FeedURL = url
	feed.ProxyURL = r.ProxyURL
	feed.WithCategoryID(r.CategoryID)
	feed.ContentChanged(body)
	feed.CheckedNow()

	processor.ProcessFeedEntries(ctx, store, feed, userID, true)

	if err := store.CreateFeed(ctx, feed); err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.database_error", err)
	}
	logging.FromContext(ctx).Debug("Created feed")

	icon.NewIconChecker(store, feed).UpdateOrCreateFeedIcon(ctx)
	return feed, nil
}
