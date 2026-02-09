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
	"miniflux.app/v2/internal/sites"
	"miniflux.app/v2/internal/storage"
	"miniflux.app/v2/internal/template"
)

var ErrBadFeed = errors.New("reader/processor: bad feed")

func UpdateEntry(user *model.User, entry *model.Entry) error {
	p := FeedProcessor{feed: entry.Feed, user: user}
	return p.UpdateEntry(entry, entry.URL)
}

type FeedProcessor struct {
	store *storage.Storage
	feed  *model.Feed
	user  *model.User
	force bool

	skipAgeFilter bool
	userByIDFunc  storage.UserByIDFunc
	templates     *template.Engine
}

func New(store *storage.Storage, feed *model.Feed, t *template.Engine,
	opts ...Option,
) *FeedProcessor {
	self := &FeedProcessor{store: store, feed: feed, templates: t}
	for _, fn := range opts {
		fn(self)
	}

	if self.userByIDFunc == nil {
		self.userByIDFunc = store.UserByID
	}
	return self
}

func (self *FeedProcessor) WithSkipAgedFilter() *FeedProcessor {
	self.skipAgeFilter = true
	return self
}

func (self *FeedProcessor) User() *model.User { return self.user }

// ProcessFeedEntries downloads original web page for entries and apply filters.
func (self *FeedProcessor) ProcessFeedEntries(ctx context.Context, userID int64,
	force bool,
) error {
	self.deleteAgedEntries(ctx)

	if len(self.feed.Entries) == 0 {
		logging.FromContext(ctx).Debug("FeedProcessor: skip processing",
			slog.String("reason", "all entries too old"))
		return nil
	}

	if err := self.markStoredEntries(ctx, force); err != nil {
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
	err := filter.DeleteEntries(ctx, self.user, self.feed,
		filter.WithSkipAgedFilter(self.skipAgeFilter))
	if err != nil {
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

	if t, err := self.feed.CommentsURLTemplate(); err == nil && t != nil {
		log = log.With(
			slog.String("template", self.feed.CommentsURLTemplateString()))
		log.Debug("make comment URLs from template")
		err := self.feed.Entries.MakeCommentURL(t)
		if err != nil {
			log.Error("failed make comment URLs from template",
				slog.Any("error", err))
		}
	}

	contentRewrite := rewrite.NewContentRewrite(self.feed.RewriteRules,
		self.user, self.templates)

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

		removeTracking(entry, self.feed.Hostnames()...)
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

		sites.Rewrite(ctx, entry)
		contentRewrite.Apply(ctx, entry)
		// The sanitizer should always run at the end of the process to make sure
		// unsafe HTML is filtered out.
		err := self.sanitizeEntry(entry, pageURL)
		if err != nil {
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

func (self *FeedProcessor) markStoredEntries(ctx context.Context, force bool,
) error {
	if force || len(self.feed.Entries) == 0 {
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

func removeTracking(entry *model.Entry, hostnames ...string) {
	u, err := entry.ParsedURL()
	if err != nil {
		return
	}

	sanitizer.StripTracking(u, hostnames...)
	entry.WithURL(u)
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

func (self *FeedProcessor) UpdateEntry(entry *model.Entry, pageURL string,
	opts ...sanitizer.Option,
) error {
	err := self.sanitizeEntry(entry, pageURL, opts...)
	if err != nil {
		return err
	}

	if self.user.ShowReadingTime {
		entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
			self.user.DefaultReadingSpeed, self.user.CJKReadingSpeed)
	}
	return nil
}

func (self *FeedProcessor) sanitizeEntry(entry *model.Entry, pageURL string,
	opts ...sanitizer.Option,
) error {
	sanitizeTitle(entry, self.feed)

	var parsedURL *url.URL
	if pageURL != "" {
		u, err := url.Parse(pageURL)
		if err != nil {
			return fmt.Errorf("reader/processor: parse page URL: %w", err)
		}
		parsedURL = u
	} else {
		u, err := entry.ParsedURL()
		if err != nil {
			return fmt.Errorf("reader/processor: %w", err)
		}
		parsedURL = u
	}

	entry.Content = sanitizer.SanitizeContent(entry.Content, parsedURL, opts...)
	return nil
}

func sanitizeTitle(entry *model.Entry, feed *model.Feed) string {
	if entry.Title != "" {
		entry.Title = sanitizer.StripTags(entry.Title)
	}

	if entry.Title == "" {
		title := makeTitle(entry.Content)
		if title == "" && entry.URL == "" {
			title = feed.SiteURL
		}
		entry.WithAutoTitle(title)
	}
	return entry.Title
}

func makeTitle(content string) string {
	title := strings.TrimSpace(sanitizer.StripTags(content))
	if title == "" {
		return title
	}

	const maxLen = 100
	title = strings.Join(strings.Fields(content), " ")
	if runes := []rune(title); len(runes) > maxLen {
		return strings.TrimSpace(string(runes[:maxLen])) + "…"
	}
	return title
}

// ProcessEntryWebPage downloads the entry web page and apply rewrite rules.
func ProcessEntryWebPage(ctx context.Context, feed *model.Feed,
	entry *model.Entry, user *model.User, opts ...sanitizer.Option,
) error {
	removeTracking(entry, feed.Hostnames()...)
	rewrite.RewriteEntryURL(ctx, feed, entry)

	p := FeedProcessor{feed: feed, user: user}
	pageURL, content, err := p.Scrape(ctx, entry)
	if err != nil || content == "" {
		return err
	}
	return p.UpdateEntry(entry, pageURL, opts...)
}
