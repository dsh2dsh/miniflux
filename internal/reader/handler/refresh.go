package handler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

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

const (
	notModifiedHeaders int = iota + 1
	notModifiedContent
)

func RefreshFeed(ctx context.Context, store *storage.Storage, userID,
	feedID int64, forceRefresh bool,
) *locale.LocalizedErrorWrapper {
	r := Refresh{
		store:  store,
		userID: userID,
		feedID: feedID,
		force:  forceRefresh,
	}
	return r.RefreshFeed(ctx)
}

type Refresh struct {
	store  *storage.Storage
	userID int64
	feedID int64
	force  bool

	feed *model.Feed
}

// RefreshFeed refreshes a feed.
func (self *Refresh) RefreshFeed(ctx context.Context,
) *locale.LocalizedErrorWrapper {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", self.userID), slog.Int64("feed_id", self.feedID))
	log.Debug("Begin feed refresh process",
		slog.Bool("force_refresh", self.force))

	startTime := time.Now()
	if err := self.initFeed(ctx); err != nil {
		return err
	}

	weeklyCount, lerr := self.weeklyCount(ctx)
	if lerr != nil {
		return lerr
	}

	self.feed.CheckedNow()
	self.feed.ScheduleNextCheck(weeklyCount, 0)

	resp, err := self.response(ctx)
	if err != nil {
		return locale.NewLocalizedErrorWrapper(err, "error.unable_to_parse_feed",
			err)
	}
	defer resp.Close()

	if resp.IsRateLimited() {
		self.logRateLimited(logging.WithLogger(ctx, log), resp, weeklyCount)
	}

	lerr = self.respLocalizedError(logging.WithLogger(ctx, log), resp)
	if lerr != nil {
		return lerr
	}

	lerr = self.anotherFeedURLExists(ctx, resp.EffectiveURL())
	if lerr != nil {
		return lerr
	}

	var refreshed model.FeedRefreshed
	if self.refreshAnyway(resp) {
		r, lerr := self.refreshFeed(logging.WithLogger(ctx, log), resp, weeklyCount)
		if lerr != nil {
			return lerr
		}
		refreshed = r
	} else {
		log.Debug("Feed not modified")
		refreshed.NotModified = notModifiedHeaders
	}

	if !refreshed.Refreshed {
		// Last-Modified may be updated even if ETag is not. In this case, per
		// RFC9111 sections 3.2 and 4.3.4, the stored response must be updated.
		if resp.LastModified() != "" {
			self.feed.LastModifiedHeader = resp.LastModified()
		}
	}

	self.feed.ResetErrorCounter()
	if err := self.updateFeed(ctx); err != nil {
		return err
	}

	self.logFeedRefresh(log, &refreshed, time.Since(startTime))
	return nil
}

func (self *Refresh) initFeed(ctx context.Context,
) *locale.LocalizedErrorWrapper {
	feed, err := self.store.FeedByID(ctx, self.userID, self.feedID)
	if err != nil {
		return locale.NewLocalizedErrorWrapper(err,
			"error.database_error", err)
	} else if feed == nil {
		return locale.NewLocalizedErrorWrapper(ErrFeedNotFound,
			"error.feed_not_found")
	}
	self.feed = feed
	return nil
}

func (self *Refresh) weeklyCount(ctx context.Context) (int,
	*locale.LocalizedErrorWrapper,
) {
	var weeklyCount int
	if config.Opts.PollingScheduler() == model.SchedulerEntryFrequency {
		cnt, err := self.store.WeeklyFeedEntryCount(ctx, self.userID, self.feedID)
		if err != nil {
			return 0, locale.NewLocalizedErrorWrapper(err,
				"error.database_error", err)
		}
		weeklyCount = cnt
	}
	return weeklyCount, nil
}

func (self *Refresh) response(ctx context.Context) (*fetcher.ResponseSemaphore,
	error,
) {
	f := self.feed
	r := fetcher.NewRequestBuilder().
		WithUsernameAndPassword(f.Username, f.Password).
		WithUserAgent(f.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(f.Cookie).
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(f.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		DisableHTTP2(f.DisableHTTP2)

	if !self.ignoreHTTPCache() {
		r.WithETag(f.EtagHeader).WithLastModified(f.LastModifiedHeader)
	}

	resp, err := fetcher.NewResponseSemaphore(ctx, r, f.FeedURL)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (self *Refresh) refreshAnyway(resp *fetcher.ResponseSemaphore) bool {
	return self.ignoreHTTPCache() ||
		resp.IsModified(self.feed.EtagHeader, self.feed.LastModifiedHeader)
}

func (self *Refresh) ignoreHTTPCache() bool {
	return self.feed.IgnoreHTTPCache || self.force
}

func (self *Refresh) logRateLimited(ctx context.Context,
	resp *fetcher.ResponseSemaphore, weeklyCount int,
) {
	retryDelaySeconds := resp.ParseRetryDelay()
	refreshDelay := retryDelaySeconds / 60
	nextCheck := self.feed.ScheduleNextCheck(weeklyCount, refreshDelay)

	logging.FromContext(ctx).Warn("Feed is rate limited",
		slog.String("feed_url", self.feed.FeedURL),
		slog.Int("retry_delay_in_seconds", retryDelaySeconds),
		slog.Int("refresh_delay_in_minutes", refreshDelay),
		slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
		slog.Time("new_next_check_at", self.feed.NextCheckAt))
}

func (self *Refresh) respLocalizedError(ctx context.Context,
	resp *fetcher.ResponseSemaphore,
) *locale.LocalizedErrorWrapper {
	if lerr := resp.LocalizedError(); lerr != nil {
		logging.FromContext(ctx).Warn("Unable to fetch feed",
			slog.String("feed_url", self.feed.FeedURL),
			slog.Any("error", lerr))
		return self.incFeedError(ctx, lerr)
	}
	return nil
}

func (self *Refresh) incFeedError(ctx context.Context,
	lerr *locale.LocalizedErrorWrapper,
) *locale.LocalizedErrorWrapper {
	user, err := self.store.UserByID(ctx, self.userID)
	if err != nil {
		return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
	}
	self.feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
	if err := self.store.IncFeedError(ctx, self.feed); err != nil {
		return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
	}
	return lerr
}

func (self *Refresh) anotherFeedURLExists(ctx context.Context, url string,
) *locale.LocalizedErrorWrapper {
	if self.store.AnotherFeedURLExists(ctx, self.userID, self.feedID, url) {
		lerr := locale.NewLocalizedErrorWrapper(ErrDuplicatedFeed,
			"error.duplicated_feed")
		return self.incFeedError(ctx, lerr)
	}
	return nil
}

func (self *Refresh) refreshFeed(ctx context.Context,
	resp *fetcher.ResponseSemaphore, weeklyCount int,
) (refreshed model.FeedRefreshed, _ *locale.LocalizedErrorWrapper) {
	log := logging.FromContext(ctx)
	log.Debug("Feed modified",
		slog.String("etag_header", self.feed.EtagHeader),
		slog.String("last_modified_header", self.feed.LastModifiedHeader))

	body, lerr := resp.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if lerr != nil {
		log.Warn("Unable to fetch feed",
			slog.String("feed_url", self.feed.FeedURL),
			slog.Any("error", lerr))
		return refreshed, lerr
	}
	resp.Close()

	if !self.feed.ContentChanged(body) && !self.force {
		refreshed.NotModified = notModifiedContent
		return refreshed, nil
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
		return refreshed, self.incFeedError(ctx, lerr)
	}

	// Use the RSS TTL value, or the Cache-Control or Expires HTTP headers if
	// available. Otherwise, we use the default value from the configuration (min
	// interval parameter).
	ttl := remoteFeed.TTL
	cacheControl := resp.CacheControlMaxAgeInMinutes()
	expires := resp.ExpiresInMinutes()
	refreshDelay := max(ttl, cacheControl, expires)

	// Set the next check at with updated arguments.
	nextCheck := self.feed.ScheduleNextCheck(weeklyCount, refreshDelay)

	log.Debug("Updated next check date",
		slog.String("feed_url", self.feed.FeedURL),
		slog.Int("feed_ttl_minutes", ttl),
		slog.Int("cache_control_max_age_in_minutes", cacheControl),
		slog.Int("expires_in_minutes", expires),
		slog.Int("refresh_delay_in_minutes", refreshDelay),
		slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
		slog.Time("new_next_check_at", self.feed.NextCheckAt))

	self.feed.Entries = remoteFeed.Entries
	processor.ProcessFeedEntries(ctx, self.store, self.feed, self.userID,
		self.force)

	// We don't update existing entries when the crawler is enabled (we crawl
	// only inexisting entries). Unless it is forced to refresh.
	update := self.force || !self.feed.Crawler
	refreshed, err = self.store.RefreshFeedEntries(ctx, self.userID, self.feedID,
		self.feed.Entries, update)
	if err != nil {
		return refreshed, self.incFeedError(ctx,
			locale.NewLocalizedErrorWrapper(err, "error.database_error", err))
	}

	self.pushIntegrations(logging.WithLogger(ctx, log), refreshed.CreatedEntries)

	self.feed.EtagHeader = resp.ETag()
	self.feed.LastModifiedHeader = resp.LastModified()
	if self.force {
		self.feed.IconURL = remoteFeed.IconURL
		icon.NewIconChecker(self.store, self.feed).UpdateOrCreateFeedIcon(ctx)
	}

	refreshed.Refreshed = true
	return refreshed, nil
}

func (self *Refresh) pushIntegrations(ctx context.Context,
	entries model.Entries,
) {
	if len(entries) == 0 {
		return
	}

	integrations, err := self.store.Integration(ctx, self.userID)
	if err != nil {
		logging.FromContext(ctx).Error(
			"Fetching integrations failed; the refresh process will go on, but no integrations will run this time",
			slog.Any("error", err))
		return
	} else if integrations == nil {
		return
	}

	integration.PushEntries(self.feed, entries, integrations)
}

func (self *Refresh) updateFeed(ctx context.Context,
) *locale.LocalizedErrorWrapper {
	if err := self.store.UpdateFeed(ctx, self.feed); err != nil {
		lerr := locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		user, err := self.store.UserByID(ctx, self.userID)
		if err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		self.feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
		if err := self.store.IncFeedError(ctx, self.feed); err != nil {
			return locale.NewLocalizedErrorWrapper(err, "error.database_error", err)
		}
		return lerr
	}
	return nil
}

func (self *Refresh) logFeedRefresh(log *slog.Logger,
	refreshed *model.FeedRefreshed, elapsed time.Duration,
) {
	var msg string
	switch {
	case refreshed.Refreshed:
		msg = "Feed refreshed"
		log = log.With(
			slog.Int("updated", len(refreshed.UpdatedEntires)),
			slog.Int("created", len(refreshed.CreatedEntries)),
			slog.Duration("storage_elapsed", refreshed.StorageElapsed))
	case refreshed.NotModified == notModifiedHeaders:
		msg = "Response not modified"
		log = log.With(
			slog.String("etag_header", self.feed.EtagHeader),
			slog.String("last_modified_header", self.feed.LastModifiedHeader))
	case refreshed.NotModified == notModifiedContent:
		msg = "Content not modified"
		log = log.With(
			slog.Uint64("size", self.feed.Extra.Size),
			slog.String("hash", strconv.FormatUint(self.feed.Extra.Hash, 16)))
	default:
		msg = "Feed not refreshed with unknown reason"
	}

	log.Info(msg,
		slog.Duration("elapsed", elapsed),
		slog.String("feed_url", self.feed.FeedURL))
}
