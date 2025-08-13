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

// ProcessFeedEntries downloads original web page for entries and apply filters.
func ProcessFeedEntries(ctx context.Context, store *storage.Storage,
	feed *model.Feed, userID int64, force bool,
) error {
	log := logging.FromContext(ctx)
	user, err := store.UserByID(ctx, userID)
	if err != nil {
		log.Error(err.Error(), slog.Int64("user_id", userID))
		return fmt.Errorf("reader/processor: fetch user id=%v: %w",
			userID, err)
	}

	if err := filter.DeleteEntries(ctx, user, feed); err != nil {
		log.Debug("entries filter completed with error", slog.Any("error", err))
		return fmt.Errorf("%w: delete filtered entries: %w", ErrBadFeed, err)
	} else if len(feed.Entries) == 0 {
		log.Debug("all entries deleted, nothing left")
		return nil
	}
	log.Debug("process filtered entries", slog.Int("entries", len(feed.Entries)))

	// process older entries first
	slices.Reverse(feed.Entries)

	if feed.Crawler && !force {
		if err := markStoredEntries(ctx, store, feed); err != nil {
			log.Error(err.Error(), slog.Int("entries", len(feed.Entries)))
			return fmt.Errorf(
				"internal/reader/processor: find known feed entries: %w", err)
		}
	}

	// The errors are handled in RemoveTrackingParameters.
	feedURL, _ := url.Parse(feed.FeedURL)
	siteURL, _ := url.Parse(feed.SiteURL)

	for _, entry := range feed.Entries {
		log := log.With(
			slog.Int64("user_id", user.ID),
			slog.GroupAttrs("entry",
				slog.Bool("stored", entry.Stored()),
				slog.String("hash", entry.Hash),
				slog.String("url", entry.URL),
				slog.String("title", entry.Title)),
			slog.GroupAttrs("feed",
				slog.Int64("id", feed.ID),
				slog.String("url", feed.FeedURL)))
		log.Debug("Processing entry")

		removeTracking(feedURL, siteURL, entry)
		rewrite.RewriteEntryURL(ctx, feed, entry)

		var pageURL string
		if feed.Crawler && (force || !entry.Stored()) {
			log := log.With(slog.Bool("force_refresh", force))
			log.Debug("Scraping entry")

			scrapedURL, _, err := scrape(ctx, feed, entry)
			if err != nil {
				log.Warn("Unable scrape entry", slog.Any("error", err))
				return fmt.Errorf("%w: scrape entry: %w", ErrBadFeed, err)
			} else if scrapedURL != "" {
				pageURL = scrapedURL
			}
		}

		rewrite.ApplyContentRewriteRules(entry, feed.RewriteRules)
		// The sanitizer should always run at the end of the process to make sure
		// unsafe HTML is filtered out.
		entry.Title = sanitizer.StripTags(entry.Title)
		if err := sanitizeEntry(entry, pageURL); err != nil {
			return fmt.Errorf("%w: %w", ErrBadFeed, err)
		}
		updateEntryReadingTime(ctx, store, feed, entry, !entry.Stored(), user)
	}

	if user.ShowReadingTime && shouldFetchYouTubeWatchTimeInBulk() {
		fetchYouTubeWatchTimeInBulk(feed.Entries)
	}
	return nil
}

func markStoredEntries(ctx context.Context, store *storage.Storage,
	feed *model.Feed,
) error {
	if len(feed.Entries) == 0 {
		return nil
	}

	entries := make(map[string]*model.Entry, len(feed.Entries))
	hashes := make([]string, len(feed.Entries))
	for i, e := range feed.Entries {
		entries[e.Hash] = e
		hashes[i] = e.Hash
	}

	knownHashes, err := store.KnownEntryHashes(ctx, feed.ID, hashes)
	if err != nil {
		return fmt.Errorf("fetch list of known entry hashes: %w", err)
	}

	for _, hash := range knownHashes {
		entries[hash].MarkStored()
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

func scrape(ctx context.Context, feed *model.Feed, entry *model.Entry,
) (string, string, error) {
	startTime := time.Now()
	builder := fetcher.NewRequestFeed(feed).WithContext(ctx)

	baseURL, content, err := scraper.ScrapeWebsite(builder, entry.URL,
		feed.ScraperRules)
	if config.Opts.HasMetricsCollector() {
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

// ProcessEntryWebPage downloads the entry web page and apply rewrite rules.
func ProcessEntryWebPage(ctx context.Context, feed *model.Feed,
	entry *model.Entry, user *model.User,
) error {
	rewrite.RewriteEntryURL(ctx, feed, entry)
	baseURL, content, err := scrape(ctx, feed, entry)
	if err != nil || content == "" {
		return err
	}

	if user.ShowReadingTime {
		entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
			user.DefaultReadingSpeed, user.CJKReadingSpeed)
	}

	rewrite.ApplyContentRewriteRules(entry, entry.Feed.RewriteRules)

	if err := sanitizeEntry(entry, baseURL); err != nil {
		return err
	}
	return nil
}

func sanitizeEntry(entry *model.Entry, pageURL string) error {
	if pageURL == "" {
		pageURL = entry.URL
	}

	u, err := url.Parse(pageURL)
	if err != nil {
		return fmt.Errorf("parse entry URL: %w", err)
	}

	entry.Content = sanitizer.SanitizeContent(entry.Content, u)
	return nil
}
