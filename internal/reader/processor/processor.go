// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/filter"
	"miniflux.app/v2/internal/reader/readingtime"
	"miniflux.app/v2/internal/reader/rewrite"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/reader/scraper"
	"miniflux.app/v2/internal/storage"
)

var ErrBadFeed = errors.New("reader/processor: bad feed")

type FeedProcessor struct {
	store *storage.Storage
	feed  *model.Feed
	user  *model.User
	force bool

	skipAgeFilter bool
	userByIDFunc  storage.UserByIDFunc
}

func New(store *storage.Storage, feed *model.Feed, opts ...Option,
) *FeedProcessor {
	self := &FeedProcessor{store: store, feed: feed}
	for _, fn := range opts {
		fn(self)
	}

	if self.userByIDFunc == nil {
		self.userByIDFunc = store.UserByID
	}
	return self
}

// ProcessFeedEntries downloads original web page for entries and apply filters.
func ProcessFeedEntries(ctx context.Context, store *storage.Storage,
	feed *model.Feed, userID int64, force bool,
) error {
	return New(store, feed).ProcessFeedEntries(ctx, userID, force)
}

func (self *FeedProcessor) WithSkipAgedFilter() *FeedProcessor {
	self.skipAgeFilter = true
	return self
}

func (self *FeedProcessor) User() *model.User { return self.user }

func (self *FeedProcessor) ProcessFeedEntries(ctx context.Context, userID int64,
	force bool,
) error {
	self.deleteAgedEntries(ctx)

	if len(self.feed.Entries) == 0 {
		logging.FromContext(ctx).Debug("FeedProcessor: skip processing",
			slog.String("reason", "all entries too old"))
		return nil
	}

	if err := self.markStoredEntries(ctx); err != nil {
		logging.FromContext(ctx).Error("Unable mark stored entries",
			slog.Int("entries", len(self.feed.Entries)),
			slog.Any("error", err))
		return fmt.Errorf("reader/processor: mark stored entries: %w", err)
	}

	if len(self.feed.Entries) == 0 {
		logging.FromContext(ctx).Debug("FeedProcessor: skip processing",
			slog.String("reason", "all entries already stored"))
		return nil
	}

	if err := self.init(ctx, userID, force); err != nil {
		return err
	}
	return self.process(ctx)
}

func (self *FeedProcessor) init(ctx context.Context, userID int64, force bool,
) error {
	user, err := self.userByIDFunc(ctx, userID)
	if err != nil {
		return fmt.Errorf("reader/processor: fetch user id=%v: %w", userID, err)
	}

	self.user, self.force = user, force
	return nil
}

func (self *FeedProcessor) process(ctx context.Context) error {
	log := logging.FromContext(ctx)
	if err := filter.DeleteEntries(ctx, self.user, self.feed); err != nil {
		log.Debug("entries filter completed with error", slog.Any("error", err))
		return fmt.Errorf("%w: delete filtered entries: %w", ErrBadFeed, err)
	} else if len(self.feed.Entries) == 0 {
		log.Debug("all entries deleted, nothing left")
		return nil
	}
	log.Debug("process filtered entries",
		slog.Int("entries", len(self.feed.Entries)))

	// process older entries first
	slices.Reverse(self.feed.Entries)

	// The errors are handled in RemoveTrackingParameters.
	feedURL, _ := url.Parse(self.feed.FeedURL)
	siteURL, _ := url.Parse(self.feed.SiteURL)

	contentRewrite := rewrite.NewContentRewrite(self.feed.RewriteRules)

	for _, entry := range self.feed.Entries {
		log := log.With(
			slog.Int64("user_id", self.user.ID),
			slog.GroupAttrs("entry",
				slog.Bool("stored", entry.Stored()),
				slog.String("hash", entry.Hash),
				slog.String("url", entry.URL),
				slog.String("title", entry.Title)),
			slog.GroupAttrs("feed",
				slog.Int64("id", self.feed.ID),
				slog.String("url", self.feed.FeedURL)))
		log.Debug("Processing entry")

		removeTracking(feedURL, siteURL, entry)
		rewrite.RewriteEntryURL(ctx, self.feed, entry)

		var pageURL string
		if self.feed.Crawler && (self.force || !entry.Stored()) {
			log := log.With(slog.Bool("force_refresh", self.force))
			log.Debug("Scraping entry")

			scrapedURL, _, err := self.Scrape(ctx, entry)
			if err != nil {
				log.Warn("Unable scrape entry", slog.Any("error", err))
				return fmt.Errorf("%w: scrape entry: %w", ErrBadFeed, err)
			} else if scrapedURL != "" {
				pageURL = scrapedURL
			}
		}

		contentRewrite.Apply(ctx, entry)
		// The sanitizer should always run at the end of the process to make sure
		// unsafe HTML is filtered out.
		if err := self.sanitizeEntry(entry, pageURL); err != nil {
			return fmt.Errorf("%w: %w", ErrBadFeed, err)
		}
		updateEntryReadingTime(ctx, self.store, self.feed, entry, !entry.Stored(),
			self.user)
	}

	if self.user.ShowReadingTime && shouldFetchYouTubeWatchTimeInBulk() {
		fetchYouTubeWatchTimeInBulk(self.feed.Entries)
	}
	return nil
}

func (self *FeedProcessor) deleteAgedEntries(ctx context.Context) {
	if !self.skipAgeFilter {
		filter.DeleteAgedEntries(ctx, self.feed)
	}
}

func (self *FeedProcessor) markStoredEntries(ctx context.Context) error {
	if len(self.feed.Entries) == 0 {
		return nil
	}

	storedEntries, err := self.store.KnownEntryHashes(ctx, self.feed.ID,
		self.feed.Entries.Hashes())
	if err != nil {
		return fmt.Errorf("fetch list of known entry hashes: %w", err)
	}

	entries := self.feed.Entries.ByHash()
	for i := range storedEntries {
		stored := &storedEntries[i]
		e := entries[stored.Hash]
		if !e.Date.After(stored.Date) {
			e.MarkStored()
		}
	}
	return nil
}

func removeTracking(feedURL, siteURL *url.URL, entry *model.Entry) {
	u, err := url.Parse(entry.URL)
	if err != nil {
		return
	}
	sanitizer.StripTracking(u, feedURL.Hostname(), siteURL.Hostname())
	entry.URL = u.String()
}

func (self *FeedProcessor) Scrape(ctx context.Context, entry *model.Entry,
) (string, string, error) {
	startTime := time.Now()
	builder := fetcher.NewRequestFeed(self.feed).WithContext(ctx)

	logging.FromContext(ctx).Info("Fetch original content",
		slog.String("url", entry.URL))

	baseURL, content, err := scraper.ScrapeWebsite(ctx, builder, entry.URL,
		self.feed.ScraperRules)
	if config.HasMetricsCollector() {
		status := "success"
		if err != nil {
			status = "error"
		}
		metric.ScraperRequestDuration.
			WithLabelValues(status).
			Observe(time.Since(startTime).Seconds())
	}

	if err != nil {
		return "", "", err
	} else if content != "" {
		// We replace the entry content only if the scraper doesn't return any
		// error.
		entry.Content = content
	}
	return baseURL, content, nil
}

func (self *FeedProcessor) sanitizeEntry(entry *model.Entry, pageURL string) error {
	if pageURL == "" {
		pageURL = entry.URL
	}
	u, err := url.Parse(pageURL)
	if err != nil {
		return fmt.Errorf("reader/processor: parse entry URL: %w", err)
	}

	entry.Title = sanitizeTitle(entry, self.feed)
	entry.Content = sanitizer.SanitizeContent(entry.Content, u)
	return nil
}

func sanitizeTitle(entry *model.Entry, feed *model.Feed) string {
	if entry.Title != "" {
		return sanitizer.StripTags(entry.Title)
	}

	title := strings.TrimSpace(sanitizer.StripTags(entry.Content))
	if title == "" {
		return feed.SiteURL
	}

	const maxLen = 100
	title = strings.Join(strings.Fields(title), " ")
	if runes := []rune(title); len(runes) > maxLen {
		return strings.TrimSpace(string(runes[:maxLen])) + "…"
	}
	return title
}

// ProcessEntryWebPage downloads the entry web page and apply rewrite rules.
func ProcessEntryWebPage(ctx context.Context, feed *model.Feed,
	entry *model.Entry, user *model.User,
) error {
	// The errors are handled in RemoveTrackingParameters.
	feedURL, _ := url.Parse(feed.FeedURL)
	siteURL, _ := url.Parse(feed.SiteURL)
	removeTracking(feedURL, siteURL, entry)
	rewrite.RewriteEntryURL(ctx, feed, entry)

	p := FeedProcessor{feed: feed, user: user}
	pageURL, content, err := p.Scrape(ctx, entry)
	if err != nil || content == "" {
		return err
	}

	if err := p.sanitizeEntry(entry, pageURL); err != nil {
		return err
	}

	if user.ShowReadingTime {
		entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
			user.DefaultReadingSpeed, user.CJKReadingSpeed)
	}
	return nil
}
