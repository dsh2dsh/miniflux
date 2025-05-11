// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler // import "miniflux.app/v2/internal/reader/handler"

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strconv"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/integration"
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
	feedCreationRequest *model.FeedCreationRequestFromSubscriptionDiscovery,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	slog.Debug("Begin feed creation process from subscription discovery",
		slog.Int64("user_id", userID),
		slog.String("feed_url", feedCreationRequest.FeedURL),
		slog.String("proxy_url", feedCreationRequest.ProxyURL),
	)

	if !store.CategoryIDExists(ctx, userID, feedCreationRequest.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound, "error.category_not_found")
	}

	if store.FeedURLExists(ctx, userID, feedCreationRequest.FeedURL) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed, "error.duplicated_feed")
	}

	subscription, parseErr := parser.ParseFeed(feedCreationRequest.FeedURL, feedCreationRequest.Content)
	if parseErr != nil {
		return nil, locale.NewLocalizedErrorWrapper(parseErr, "error.unable_to_parse_feed", parseErr)
	}

	subscription.UserID = userID
	subscription.UserAgent = feedCreationRequest.UserAgent
	subscription.Cookie = feedCreationRequest.Cookie
	subscription.Username = feedCreationRequest.Username
	subscription.Password = feedCreationRequest.Password
	subscription.Crawler = feedCreationRequest.Crawler
	subscription.Disabled = feedCreationRequest.Disabled
	subscription.IgnoreHTTPCache = feedCreationRequest.IgnoreHTTPCache
	subscription.AllowSelfSignedCertificates = feedCreationRequest.AllowSelfSignedCertificates
	subscription.FetchViaProxy = feedCreationRequest.FetchViaProxy
	subscription.ScraperRules = feedCreationRequest.ScraperRules
	subscription.RewriteRules = feedCreationRequest.RewriteRules
	subscription.BlocklistRules = feedCreationRequest.BlocklistRules
	subscription.KeeplistRules = feedCreationRequest.KeeplistRules
	subscription.UrlRewriteRules = feedCreationRequest.UrlRewriteRules
	subscription.EtagHeader = feedCreationRequest.ETag
	subscription.LastModifiedHeader = feedCreationRequest.LastModified
	subscription.FeedURL = feedCreationRequest.FeedURL
	subscription.DisableHTTP2 = feedCreationRequest.DisableHTTP2
	subscription.WithCategoryID(feedCreationRequest.CategoryID)
	subscription.ProxyURL = feedCreationRequest.ProxyURL
	subscription.CheckedNow()

	processor.ProcessFeedEntries(ctx, store, subscription, userID, true)

	if storeErr := store.CreateFeed(ctx, subscription); storeErr != nil {
		return nil, locale.NewLocalizedErrorWrapper(storeErr, "error.database_error", storeErr)
	}

	slog.Debug("Created feed",
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", subscription.ID),
		slog.String("feed_url", subscription.FeedURL),
	)

	requestBuilder := fetcher.NewRequestBuilder()
	requestBuilder.WithUsernameAndPassword(feedCreationRequest.Username, feedCreationRequest.Password)
	requestBuilder.WithUserAgent(feedCreationRequest.UserAgent, config.Opts.HTTPClientUserAgent())
	requestBuilder.WithCookie(feedCreationRequest.Cookie)
	requestBuilder.WithTimeout(config.Opts.HTTPClientTimeout())
	requestBuilder.WithProxyRotator(proxyrotator.ProxyRotatorInstance)
	requestBuilder.WithCustomFeedProxyURL(feedCreationRequest.ProxyURL)
	requestBuilder.WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL())
	requestBuilder.UseCustomApplicationProxyURL(feedCreationRequest.FetchViaProxy)
	requestBuilder.IgnoreTLSErrors(feedCreationRequest.AllowSelfSignedCertificates)
	requestBuilder.DisableHTTP2(feedCreationRequest.DisableHTTP2)

	icon.NewIconChecker(store, subscription).UpdateOrCreateFeedIcon(ctx)

	return subscription, nil
}

// CreateFeed fetch, parse and store a new feed.
func CreateFeed(ctx context.Context, store *storage.Storage, userID int64,
	feedCreationRequest *model.FeedCreationRequest,
) (*model.Feed, *locale.LocalizedErrorWrapper) {
	slog.Debug("Begin feed creation process",
		slog.Int64("user_id", userID),
		slog.String("feed_url", feedCreationRequest.FeedURL),
		slog.String("proxy_url", feedCreationRequest.ProxyURL),
	)

	if !store.CategoryIDExists(ctx, userID, feedCreationRequest.CategoryID) {
		return nil, locale.NewLocalizedErrorWrapper(ErrCategoryNotFound, "error.category_not_found")
	}

	requestBuilder := fetcher.NewRequestBuilder()
	requestBuilder.WithUsernameAndPassword(feedCreationRequest.Username, feedCreationRequest.Password)
	requestBuilder.WithUserAgent(feedCreationRequest.UserAgent, config.Opts.HTTPClientUserAgent())
	requestBuilder.WithCookie(feedCreationRequest.Cookie)
	requestBuilder.WithTimeout(config.Opts.HTTPClientTimeout())
	requestBuilder.WithProxyRotator(proxyrotator.ProxyRotatorInstance)
	requestBuilder.WithCustomFeedProxyURL(feedCreationRequest.ProxyURL)
	requestBuilder.WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL())
	requestBuilder.UseCustomApplicationProxyURL(feedCreationRequest.FetchViaProxy)
	requestBuilder.IgnoreTLSErrors(feedCreationRequest.AllowSelfSignedCertificates)
	requestBuilder.DisableHTTP2(feedCreationRequest.DisableHTTP2)

	responseHandler := fetcher.NewResponseHandler(requestBuilder.ExecuteRequest(feedCreationRequest.FeedURL))
	defer responseHandler.Close()

	if localizedError := responseHandler.LocalizedError(); localizedError != nil {
		slog.Warn("Unable to fetch feed", slog.String("feed_url", feedCreationRequest.FeedURL), slog.Any("error", localizedError.Error()))
		return nil, localizedError
	}

	responseBody, localizedError := responseHandler.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if localizedError != nil {
		slog.Warn("Unable to fetch feed", slog.String("feed_url", feedCreationRequest.FeedURL), slog.Any("error", localizedError.Error()))
		return nil, localizedError
	}

	if store.FeedURLExists(ctx, userID, responseHandler.EffectiveURL()) {
		return nil, locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed, "error.duplicated_feed")
	}

	subscription, parseErr := parser.ParseFeed(responseHandler.EffectiveURL(), bytes.NewReader(responseBody))
	if parseErr != nil {
		return nil, locale.NewLocalizedErrorWrapper(parseErr, "error.unable_to_parse_feed", parseErr)
	}

	subscription.UserID = userID
	subscription.UserAgent = feedCreationRequest.UserAgent
	subscription.Cookie = feedCreationRequest.Cookie
	subscription.Username = feedCreationRequest.Username
	subscription.Password = feedCreationRequest.Password
	subscription.Crawler = feedCreationRequest.Crawler
	subscription.Disabled = feedCreationRequest.Disabled
	subscription.IgnoreHTTPCache = feedCreationRequest.IgnoreHTTPCache
	subscription.AllowSelfSignedCertificates = feedCreationRequest.AllowSelfSignedCertificates
	subscription.DisableHTTP2 = feedCreationRequest.DisableHTTP2
	subscription.FetchViaProxy = feedCreationRequest.FetchViaProxy
	subscription.ScraperRules = feedCreationRequest.ScraperRules
	subscription.RewriteRules = feedCreationRequest.RewriteRules
	subscription.BlocklistRules = feedCreationRequest.BlocklistRules
	subscription.KeeplistRules = feedCreationRequest.KeeplistRules
	subscription.UrlRewriteRules = feedCreationRequest.UrlRewriteRules
	subscription.HideGlobally = feedCreationRequest.HideGlobally
	subscription.EtagHeader = responseHandler.ETag()
	subscription.LastModifiedHeader = responseHandler.LastModified()
	subscription.FeedURL = responseHandler.EffectiveURL()
	subscription.ProxyURL = feedCreationRequest.ProxyURL
	subscription.WithCategoryID(feedCreationRequest.CategoryID)
	subscription.ContentChanged(responseBody)
	subscription.CheckedNow()

	processor.ProcessFeedEntries(ctx, store, subscription, userID, true)

	if storeErr := store.CreateFeed(ctx, subscription); storeErr != nil {
		return nil, locale.NewLocalizedErrorWrapper(storeErr, "error.database_error", storeErr)
	}

	slog.Debug("Created feed",
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", subscription.ID),
		slog.String("feed_url", subscription.FeedURL),
	)

	icon.NewIconChecker(store, subscription).UpdateOrCreateFeedIcon(ctx)

	return subscription, nil
}

// RefreshFeed refreshes a feed.
func RefreshFeed(ctx context.Context, store *storage.Storage, userID,
	feedID int64, forceRefresh bool,
) *locale.LocalizedErrorWrapper {
	slog.Debug("Begin feed refresh process",
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", feedID),
		slog.Bool("force_refresh", forceRefresh))

	feed, err := store.FeedByID(ctx, userID, feedID)
	if err != nil {
		return locale.NewLocalizedErrorWrapper(err,
			"error.database_error", err)
	} else if feed == nil {
		return locale.NewLocalizedErrorWrapper(ErrFeedNotFound,
			"error.feed_not_found")
	}

	weeklyCount := 0
	refreshDelay := 0
	if config.Opts.PollingScheduler() == model.SchedulerEntryFrequency {
		cnt, err := store.WeeklyFeedEntryCount(ctx, userID, feedID)
		if err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		weeklyCount = cnt
	}

	feed.CheckedNow()
	feed.ScheduleNextCheck(weeklyCount, refreshDelay)

	r := fetcher.NewRequestBuilder().
		WithUsernameAndPassword(feed.Username, feed.Password).
		WithUserAgent(feed.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(feed.Cookie).
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(feed.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(feed.FetchViaProxy).
		IgnoreTLSErrors(feed.AllowSelfSignedCertificates).
		DisableHTTP2(feed.DisableHTTP2)

	ignoreHTTPCache := feed.IgnoreHTTPCache || forceRefresh
	if !ignoreHTTPCache {
		r.WithETag(feed.EtagHeader).WithLastModified(feed.LastModifiedHeader)
	}

	resp := fetcher.NewResponseHandler(r.ExecuteRequest(feed.FeedURL))
	defer resp.Close()

	if resp.IsRateLimited() {
		retryDelaySeconds := resp.ParseRetryDelay()
		refreshDelay = retryDelaySeconds / 60
		nextCheck := feed.ScheduleNextCheck(weeklyCount, refreshDelay)
		slog.Warn("Feed is rate limited",
			slog.String("feed_url", feed.FeedURL),
			slog.Int("retry_delay_in_seconds", retryDelaySeconds),
			slog.Int("refresh_delay_in_minutes", refreshDelay),
			slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
			slog.Time("new_next_check_at", feed.NextCheckAt))
	}

	if lerr := resp.LocalizedError(); lerr != nil {
		slog.Warn("Unable to fetch feed",
			slog.Int64("user_id", userID),
			slog.Int64("feed_id", feedID),
			slog.String("feed_url", feed.FeedURL),
			slog.Any("error", lerr))
		user, err := store.UserByID(ctx, userID)
		if err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := store.UpdateFeedError(ctx, feed); err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		return lerr
	}

	if store.AnotherFeedURLExists(ctx, userID, feed.ID, resp.EffectiveURL()) {
		lerr := locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
		user, err := store.UserByID(ctx, userID)
		if err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := store.UpdateFeedError(ctx, feed); err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		return lerr
	}

	refreshAnyway := ignoreHTTPCache ||
		resp.IsModified(feed.EtagHeader, feed.LastModifiedHeader)
	var modified bool
	if refreshAnyway {
		ok, lerr := refreshFeed(ctx, store, userID, feed, resp, weeklyCount,
			forceRefresh)
		if lerr != nil {
			return lerr
		}
		modified = ok
	}

	if !modified {
		slog.Debug("Feed not modified",
			slog.Int64("user_id", userID), slog.Int64("feed_id", feedID))
		// Last-Modified may be updated even if ETag is not. In this case, per
		// RFC9111 sections 3.2 and 4.3.4, the stored response must be updated.
		if resp.LastModified() != "" {
			feed.LastModifiedHeader = resp.LastModified()
		}
	}
	feed.ResetErrorCounter()

	if err := store.UpdateFeed(ctx, feed); err != nil {
		lerr := locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		user, err := store.UserByID(ctx, userID)
		if err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := store.UpdateFeedError(ctx, feed); err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		return lerr
	}
	return nil
}

func refreshFeed(ctx context.Context, store *storage.Storage, userID int64,
	feed *model.Feed, resp *fetcher.ResponseHandler, weeklyCount int,
	forceRefresh bool,
) (bool, *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", userID),
		slog.Int64("feed_id", feed.ID))

	log.Debug("Feed modified",
		slog.String("etag_header", feed.EtagHeader),
		slog.String("last_modified_header", feed.LastModifiedHeader))

	body, lerr := resp.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if lerr != nil {
		log.Warn("Unable to fetch feed",
			slog.String("feed_url", feed.FeedURL), slog.Any("error", lerr))
		return false, lerr
	}

	if !feed.ContentChanged(body) && !forceRefresh {
		log.Info("Feed content not modified",
			slog.Uint64("size", feed.Extra.Size),
			slog.String("hash", strconv.FormatUint(feed.Extra.Hash, 16)),
			slog.String("feed_url", feed.FeedURL))
		return false, nil
	}

	remoteFeed, err := parser.ParseFeed(resp.EffectiveURL(),
		bytes.NewReader(body))
	if err != nil {
		lerr := locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_feed", err)
		if errors.Is(err, parser.ErrFeedFormatNotDetected) {
			lerr = locale.NewLocalizedErrorWrapper(err,
				"error.feed_format_not_detected", err)
		}

		user, err := store.UserByID(ctx, userID)
		if err != nil {
			return false, locale.NewLocalizedErrorWrapper(err,
				"error.database_error", err)
		}

		feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := store.UpdateFeedError(ctx, feed); err != nil {
			return false, locale.NewLocalizedErrorWrapper(err,
				"error.database_error", err)
		}
		return false, lerr
	}

	// Use the RSS TTL value, or the Cache-Control or Expires HTTP headers if
	// available. Otherwise, we use the default value from the configuration (min
	// interval parameter).
	ttl := remoteFeed.TTL
	cacheControl := resp.CacheControlMaxAgeInMinutes()
	expires := resp.ExpiresInMinutes()
	refreshDelay := max(ttl, cacheControl, expires)

	// Set the next check at with updated arguments.
	nextCheck := feed.ScheduleNextCheck(weeklyCount, refreshDelay)

	log.Debug("Updated next check date",
		slog.String("feed_url", feed.FeedURL),
		slog.Int("feed_ttl_minutes", ttl),
		slog.Int("cache_control_max_age_in_minutes", cacheControl),
		slog.Int("expires_in_minutes", expires),
		slog.Int("refresh_delay_in_minutes", refreshDelay),
		slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
		slog.Time("new_next_check_at", feed.NextCheckAt))

	feed.Entries = remoteFeed.Entries
	processor.ProcessFeedEntries(ctx, store, feed, userID, forceRefresh)

	// We don't update existing entries when the crawler is enabled (we crawl
	// only inexisting entries). Unless it is forced to refresh.
	update := forceRefresh || !feed.Crawler
	newEntries, err := store.RefreshFeedEntries(
		logging.WithLogger(ctx, log.With(
			slog.String("feed_url", feed.FeedURL),
			slog.Bool("update_existing", update))),
		feed.UserID, feed.ID, feed.Entries, update)
	if err != nil {
		lerr := locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		user, err := store.UserByID(ctx, userID)
		if err != nil {
			return false, locale.NewLocalizedErrorWrapper(err,
				"error.database_error", err)
		}
		feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := store.UpdateFeedError(ctx, feed); err != nil {
			return false, locale.NewLocalizedErrorWrapper(err,
				"error.database_error", err)
		}
		return false, lerr
	}

	if len(newEntries) > 0 {
		if integrations, err := store.Integration(ctx, userID); err != nil {
			log.Error(
				"Fetching integrations failed; the refresh process will go on, but no integrations will run this time",
				slog.Any("error", err))
		} else if integrations != nil {
			integration.PushEntries(feed, newEntries, integrations)
		}
	}

	feed.EtagHeader = resp.ETag()
	feed.LastModifiedHeader = resp.LastModified()
	if forceRefresh {
		feed.IconURL = remoteFeed.IconURL
		icon.NewIconChecker(store, feed).UpdateOrCreateFeedIcon(ctx)
	}
	return true, nil
}
