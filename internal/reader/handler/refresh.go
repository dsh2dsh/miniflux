package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/integration"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
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

var ErrBadFeed = errors.New("reader/handler: bad feed")

func RefreshFeed(ctx context.Context, store *storage.Storage, userID,
	feedID int64, forceRefresh bool,
) error {
	r := Refresh{
		store:  store,
		userID: userID,
		feedID: feedID,
		force:  forceRefresh,
	}

	if err := r.Refresh(ctx); err != nil {
		var lerr *locale.LocalizedErrorWrapper
		if errors.As(err, &lerr) {
			if err := r.incFeedError(ctx, lerr); err != nil {
				return err
			}
			return fmt.Errorf("%w: %w", ErrBadFeed, err)
		}
		return err
	}
	return nil
}

type Refresh struct {
	store  *storage.Storage
	userID int64
	feedID int64
	force  bool

	feed *model.Feed
}

func (self *Refresh) incFeedError(ctx context.Context,
	lerr *locale.LocalizedErrorWrapper,
) error {
	user, err := self.store.UserByID(ctx, self.userID)
	if err != nil {
		return fmt.Errorf("reader/handler: fetch user from db: %w: %w", err, lerr)
	}
	self.feed.WithTranslatedErrorMessage(lerr.Translate(user.Language))
	if err := self.store.IncFeedError(ctx, self.feed); err != nil {
		return fmt.Errorf("reader/handler: inc feed error count: %w: %w", err, lerr)
	}
	return nil
}

// Refresh refreshes a feed.
func (self *Refresh) Refresh(ctx context.Context) error {
	log := logging.FromContext(ctx).With(
		slog.Int64("user_id", self.userID),
		slog.Int64("feed_id", self.feedID))
	log.Debug("Begin feed refresh process",
		slog.Bool("force_refresh", self.force))

	ctx = withTraceStat(ctx)
	startTime := time.Now()
	if err := self.initFeed(ctx); err != nil {
		return err
	}

	self.feed.CheckedNow()
	self.feed.ScheduleNextCheck(0)

	resp, err := self.response(ctx)
	if err != nil {
		return err
	}
	defer resp.Close()

	if resp.IsRateLimited() {
		self.logRateLimited(logging.WithLogger(ctx, log), resp)
	}

	err = self.respError(logging.WithLogger(ctx, log), resp)
	if err != nil {
		return err
	}

	var refreshed *model.FeedRefreshed
	if self.refreshAnyway(resp) {
		r, err := self.refreshFeed(logging.WithLogger(ctx, log), resp)
		if err != nil {
			return err
		}
		refreshed = r
	} else {
		log.Debug("Feed not modified")
		refreshed = &model.FeedRefreshed{NotModified: notModifiedHeaders}
	}
	log.Info("Feed refreshed")

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

	self.logFeedRefreshed(logging.WithLogger(ctx, log), refreshed,
		time.Since(startTime))
	return nil
}

func withTraceStat(ctx context.Context) context.Context {
	if t := storage.TraceStatFrom(ctx); t != nil {
		return ctx
	}
	return storage.WithTraceStat(ctx)
}

func (self *Refresh) initFeed(ctx context.Context) error {
	feed, err := self.store.FeedByID(ctx, self.userID, self.feedID)
	if err != nil {
		return fmt.Errorf("reader/handler: get feed from db: %w", err)
	} else if feed == nil {
		return fmt.Errorf("reader/handler: %w", ErrFeedNotFound)
	}
	self.feed = feed
	return nil
}

func (self *Refresh) response(ctx context.Context) (*fetcher.ResponseSemaphore,
	error,
) {
	f := self.feed
	r := fetcher.NewRequestBuilder().
		WithUsernameAndPassword(f.Username, f.Password).
		WithUserAgent(f.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(f.Cookie).
		WithCustomFeedProxyURL(f.ProxyURL).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		DisableHTTP2(f.DisableHTTP2)

	if !self.ignoreHTTPCache() {
		r.WithETag(f.EtagHeader).WithLastModified(f.LastModifiedHeader)
	}

	resp, err := fetcher.NewResponseSemaphore(ctx, r, f.FeedURL)
	if err != nil {
		return nil, fmt.Errorf("reader/handler: fetch feed with semaphore: %w", err)
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
	resp *fetcher.ResponseSemaphore,
) {
	retryDelaySeconds := resp.ParseRetryDelay()
	refreshDelay := retryDelaySeconds / 60
	nextCheck := self.feed.ScheduleNextCheck(refreshDelay)

	logging.FromContext(ctx).Warn("Feed is rate limited",
		slog.String("feed_url", self.feed.FeedURL),
		slog.Int("retry_delay_in_seconds", retryDelaySeconds),
		slog.Int("refresh_delay_in_minutes", refreshDelay),
		slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
		slog.Time("new_next_check_at", self.feed.NextCheckAt))
}

func (self *Refresh) respError(ctx context.Context,
	resp *fetcher.ResponseSemaphore,
) error {
	if lerr := resp.LocalizedError(); lerr != nil {
		logging.FromContext(ctx).Warn("Unable to fetch feed",
			slog.String("feed_url", self.feed.FeedURL),
			slog.Any("error", lerr))
		return lerr
	}
	return nil
}

func (self *Refresh) refreshFeed(ctx context.Context,
	resp *fetcher.ResponseSemaphore,
) (*model.FeedRefreshed, error) {
	log := logging.FromContext(ctx)
	log.Info("Feed modified",
		slog.String("etag_header", self.feed.EtagHeader),
		slog.String("last_modified_header", self.feed.LastModifiedHeader))

	body, lerr := resp.ReadBody(config.Opts.HTTPClientMaxBodySize())
	if lerr != nil {
		log.Warn("Unable to fetch feed body",
			slog.String("feed_url", self.feed.FeedURL),
			slog.Any("error", lerr))
		return nil, lerr
	}
	log.Info("Read feed body", slog.Int("size", len(body)))
	resp.Close()
	log.Info("Feed body closed")

	if !self.feed.ContentChanged(body) && !self.force {
		return &model.FeedRefreshed{NotModified: notModifiedContent}, nil
	}

	log.Info("Parse feed")
	remoteFeed, err := parser.ParseFeed(resp.EffectiveURL(),
		bytes.NewReader(body))
	if err != nil {
		var lerr *locale.LocalizedErrorWrapper
		if errors.Is(err, parser.ErrFeedFormatNotDetected) {
			lerr = locale.NewLocalizedErrorWrapper(err,
				"error.feed_format_not_detected", err)
		} else {
			lerr = locale.NewLocalizedErrorWrapper(err, "error.unable_to_parse_feed",
				err)
		}
		log.Warn("Unable to parse feed body",
			slog.String("feed_url", self.feed.FeedURL),
			slog.Any("error", lerr))
		return nil, lerr
	}

	// Use the RSS TTL value, or the Cache-Control or Expires HTTP headers if
	// available. Otherwise, we use the default value from the configuration (min
	// interval parameter).
	ttl := remoteFeed.TTL
	cacheControl := resp.CacheControlMaxAgeInMinutes()
	expires := resp.ExpiresInMinutes()
	refreshDelay := max(ttl, cacheControl, expires)

	// Set the next check at with updated arguments.
	nextCheck := self.feed.ScheduleNextCheck(refreshDelay)

	log.Debug("Updated next check date",
		slog.String("feed_url", self.feed.FeedURL),
		slog.Int("feed_ttl_minutes", ttl),
		slog.Int("cache_control_max_age_in_minutes", cacheControl),
		slog.Int("expires_in_minutes", expires),
		slog.Int("refresh_delay_in_minutes", refreshDelay),
		slog.Int("calculated_next_check_interval_in_minutes", nextCheck),
		slog.Time("new_next_check_at", self.feed.NextCheckAt))

	self.feed.Entries = remoteFeed.Entries
	err = processor.ProcessFeedEntries(ctx, self.store, self.feed, self.userID,
		self.force)
	if err != nil {
		if errors.Is(err, processor.ErrBadFeed) {
			return nil, locale.NewLocalizedErrorWrapper(err,
				"error.unable_to_parse_feed", err)
		}
		return nil, err
	}

	// We don't update existing entries when the crawler is enabled (we crawl
	// only inexisting entries). Unless it is forced to refresh.
	update := self.force || !self.feed.Crawler
	refreshed, err := self.store.RefreshFeedEntries(ctx, self.userID, self.feedID,
		self.feed.Entries, update)
	if err != nil {
		return nil, fmt.Errorf("reader/handler: store feed entries: %w", err)
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

	user, err := self.store.UserByID(ctx, self.userID)
	if err != nil {
		logging.FromContext(ctx).Error(
			"Fetching integrations failed; the refresh process will go on, but no integrations will run this time",
			slog.Any("error", err))
		return
	}
	integration.PushEntries(self.feed, entries, user)
}

func (self *Refresh) updateFeed(ctx context.Context) error {
	if err := self.store.UpdateFeedRuntime(ctx, self.feed); err != nil {
		return fmt.Errorf("reader/handler: update feed runtime: %w", err)
	}
	return nil
}

func (self *Refresh) logFeedRefreshed(ctx context.Context,
	refreshed *model.FeedRefreshed, elapsed time.Duration,
) {
	log := logging.FromContext(ctx)
	var msg string

	switch {
	case refreshed.Refreshed:
		msg = "Feed refreshed"
		log = log.With(
			slog.Uint64("size", self.feed.Size()),
			slog.String("hash", self.feed.HashString()))
		log = withEntriesLogAttrs(log, self.feed, refreshed)
	case refreshed.NotModified == notModifiedHeaders:
		msg = "Response not modified"
		log = log.With(
			slog.String("etag_header", self.feed.EtagHeader),
			slog.String("last_modified_header", self.feed.LastModifiedHeader))
	case refreshed.NotModified == notModifiedContent:
		msg = "Content not modified"
		log = log.With(
			slog.Uint64("size", self.feed.Size()),
			slog.String("hash", self.feed.HashString()))
	default:
		msg = "Feed not refreshed with unknown reason"
	}

	if t := storage.TraceStatFrom(ctx); t != nil && t.Queries > 0 {
		log = log.With(slog.Group("storage",
			slog.Int64("queries", t.Queries),
			slog.Duration("elapsed", t.Elapsed)))
	}

	log.Info(msg,
		slog.Duration("elapsed", elapsed),
		slog.String("feed_url", self.feed.FeedURL))
}

func withEntriesLogAttrs(log *slog.Logger, feed *model.Feed,
	refreshed *model.FeedRefreshed,
) *slog.Logger {
	filteredAttrs := make([]any, 0, 3)
	if n := feed.RemovedByAge(); n > 0 {
		filteredAttrs = append(filteredAttrs, slog.Int("age", n))
	}
	if n := feed.RemovedByFilters(); n > 0 {
		filteredAttrs = append(filteredAttrs, slog.Int("rules", n))
	}
	if n := feed.RemovedByHash(); n > 0 {
		filteredAttrs = append(filteredAttrs, slog.Int("hash", n))
	}
	if len(filteredAttrs) != 0 {
		log = log.With(slog.Group("filtered", filteredAttrs...))
	}

	entriesAttrs := make([]any, 0, 4)
	if n := len(feed.Entries); n != 0 {
		entriesAttrs = append(entriesAttrs, slog.Int("all", n))
	}
	if n := len(refreshed.UpdatedEntires); n != 0 {
		entriesAttrs = append(entriesAttrs, slog.Int("update", n))
	}
	if n := len(refreshed.CreatedEntries); n != 0 {
		entriesAttrs = append(entriesAttrs, slog.Int("create", n))
	}
	if n := refreshed.Dedups; n > 0 {
		entriesAttrs = append(entriesAttrs, slog.Int("dedup", n))
	}
	if len(entriesAttrs) != 0 {
		log = log.With(slog.Group("entries", entriesAttrs...))
	}
	return log
}
