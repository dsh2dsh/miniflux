// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler // import "miniflux.app/v2/internal/reader/handler"

import (
	"context"
	"errors"
	"log/slog"

	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/icon"
	"miniflux.app/v2/internal/reader/parser"
	"miniflux.app/v2/internal/reader/processor"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
)

var (
	ErrCategoryNotFound = errors.New("fetcher: category not found")
	ErrFeedNotFound     = errors.New("fetcher: feed not found")
	ErrDuplicatedFeed   = errors.New("fetcher: duplicated feed")
)

type Create struct {
	store     *storage.Storage
	userID    int64
	templates *template.Engine
}

func New(store *storage.Storage, userID int64, t *template.Engine) *Create {
	return &Create{store: store, userID: userID, templates: t}
}

func (self *Create) FromDiscovery(ctx context.Context,
	r *model.FeedCreationRequestFromSubscriptionDiscovery,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", self.userID),
		slog.String("feed_url", r.FeedURL),
		slog.String("proxy_url", r.ProxyURL))
	log.Debug("Begin feed creation process from subscription discovery")

	if !self.store.CategoryIDExists(ctx, self.userID, r.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound,
			"error.category_not_found")
	}

	if self.store.FeedURLExists(ctx, self.userID, r.FeedURL) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
	}

	return self.createFeed(logging.WithLogger(ctx, log), &r.FeedCreationRequest,
		r.FeedURL, r.ETag, r.LastModified, r.Content)
}

// CreateFeed fetch, parse and store a new feed.
func (self *Create) FromRequest(ctx context.Context,
	r *model.FeedCreationRequest,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", self.userID),
		slog.String("feed_url", r.FeedURL),
		slog.String("proxy_url", r.ProxyURL))
	log.Debug("Begin feed creation process")

	if !self.store.CategoryIDExists(ctx, self.userID, r.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound,
			"error.category_not_found")
	}

	resp, err := fetcher.RequestFeedCreation(r)
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

	if self.store.FeedURLExists(ctx, self.userID, resp.EffectiveURL()) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
	}

	body, lerr := resp.ReadBody()
	if lerr != nil {
		log.Warn("Unable to fetch feed", slog.Any("error", lerr))
		return nil, lerr
	}
	resp.Close()

	return self.createFeed(logging.WithLogger(ctx, log), r, resp.EffectiveURL(),
		resp.ETag(), resp.LastModified(), body)
}

func (self *Create) createFeed(ctx context.Context,
	r *model.FeedCreationRequest, url, etag, lastModified string, body []byte,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	feed, err := parser.ParseBytes(url, body)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_feed", err)
	}

	feed.UserID = self.userID
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
	feed.UrlRewriteRules = r.UrlRewriteRules
	feed.Extra.BlockFilterEntryRules = r.BlockFilterEntryRules
	feed.Extra.KeepFilterEntryRules = r.KeepFilterEntryRules
	feed.HideGlobally = r.HideGlobally
	feed.EtagHeader = etag
	feed.LastModifiedHeader = lastModified
	feed.FeedURL = url
	feed.ProxyURL = r.ProxyURL

	feed.WithBlockAuthors(r.BlockAuthors)
	feed.WithCategoryID(r.CategoryID)
	feed.ContentChanged(body)
	feed.CheckedNow()

	err = processor.New(self.store, feed, self.templates).
		WithSkipAgedFilter().
		ProcessFeedEntries(ctx, self.userID, true)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err, "", err)
	}

	if err := self.store.CreateFeed(ctx, feed); err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err, "", err)
	}
	logging.FromContext(ctx).Debug("Created feed")

	icon.NewIconChecker(self.store, feed).UpdateOrCreateFeedIcon(ctx)
	return feed, nil
}
