// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/metric"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/readingtime"
	"miniflux.app/v2/internal/reader/rewrite"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/reader/scraper"
	"miniflux.app/v2/internal/reader/urlcleaner"
	"miniflux.app/v2/internal/storage"
)

var ErrScrape = errors.New("internal/reader/processor: unable scrape entry")

var customReplaceRuleRegex = regexp.MustCompile(
	`rewrite\("([^"]+)"\|"([^"]+)"\)`)

// ProcessFeedEntries downloads original web page for entries and apply filters.
func ProcessFeedEntries(ctx context.Context, store *storage.Storage,
	feed *model.Feed, userID int64, force bool,
) error {
	log := logging.FromContext(ctx)
	user, err := store.UserByID(ctx, userID)
	if err != nil {
		log.Error(err.Error(), slog.Int64("user_id", userID))
		return fmt.Errorf("internal/reader/processor: fetch user id=%v: %w",
			userID, err)
	}

	deleteBadEntries(user, feed)
	if feed.Crawler && !force {
		if err := markStoredEntries(ctx, store, feed); err != nil {
			log.Error(err.Error(), slog.Int("entries", len(feed.Entries)))
			return fmt.Errorf(
				"internal/reader/processor: find known feed entries: %w", err)
		}
	}

	for _, entry := range feed.Entries {
		log := log.With(
			slog.Int64("user_id", user.ID),
			slog.Group("entry",
				slog.Bool("stored", entry.Stored()),
				slog.String("hash", entry.Hash),
				slog.String("url", entry.URL),
				slog.String("title", entry.Title)),
			slog.Group("feed",
				slog.Int64("id", feed.ID),
				slog.String("url", feed.FeedURL)))
		log.Debug("Processing entry")

		removeTracking(feed, entry)
		rewriteEntryURL(ctx, feed, entry)

		var pageBaseURL string
		if feed.Crawler && (force || !entry.Stored()) {
			log := log.With(slog.Bool("force_refresh", force))
			log.Debug("Scraping entry")

			scrapedURL, _, err := scrape(ctx, feed, entry)
			if err != nil {
				log.Warn("Unable scrape entry", slog.Any("error", err))
				return fmt.Errorf("%w: %w", ErrScrape, err)
			} else if scrapedURL != "" {
				pageBaseURL = scrapedURL
			}
		}

		rewrite.Rewriter(entry.URL, entry, feed.RewriteRules)
		if pageBaseURL == "" {
			pageBaseURL = entry.URL
		}

		// The sanitizer should always run at the end of the process to make sure
		// unsafe HTML is filtered out.
		entry.Content = sanitizer.SanitizeHTML(pageBaseURL, entry.Content,
			&sanitizer.SanitizerOptions{
				OpenLinksInNewTab: !user.OpenExternalLinkSameTab(),
			})
		updateEntryReadingTime(ctx, store, feed, entry, !entry.Stored(), user)
	}

	if user.ShowReadingTime && shouldFetchYouTubeWatchTimeInBulk() {
		fetchYouTubeWatchTimeInBulk(feed.Entries)
	}
	return nil
}

func deleteBadEntries(user *model.User, feed *model.Feed) {
	entries := slices.DeleteFunc(feed.Entries, func(entry *model.Entry) bool {
		return !recentEntry(entry) ||
			isBlockedEntry(feed, entry, user) ||
			!isAllowedEntry(feed, entry, user)
	})
	// process older entries first
	slices.Reverse(entries)
	feed.Entries = entries
}

func recentEntry(entry *model.Entry) bool {
	return config.Opts.FilterEntryMaxAgeDays() == 0 ||
		entry.Date.After(time.Now().AddDate(
			0, 0, -config.Opts.FilterEntryMaxAgeDays()))
}

func markStoredEntries(ctx context.Context, store *storage.Storage,
	feed *model.Feed,
) error {
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

func removeTracking(feed *model.Feed, entry *model.Entry) {
	cleanURL, err := urlcleaner.RemoveTrackingParameters(feed.FeedURL,
		feed.SiteURL, entry.URL)
	if err == nil {
		entry.URL = cleanURL
	}
}

func rewriteEntryURL(ctx context.Context, feed *model.Feed, entry *model.Entry,
) {
	if feed.UrlRewriteRules == "" {
		return
	}

	log := logging.FromContext(ctx)
	parts := customReplaceRuleRegex.FindStringSubmatch(feed.UrlRewriteRules)
	if len(parts) < 3 {
		log.Debug("Cannot find search and replace terms for replace rule",
			slog.String("entry_url", entry.URL),
			slog.Int64("feed_id", feed.ID),
			slog.String("feed_url", feed.FeedURL),
			slog.String("url_rewrite_rules", feed.UrlRewriteRules))
		return
	}

	re, err := regexp.Compile(parts[1])
	if err != nil {
		log.Error("Failed on regexp compilation",
			slog.String("url_rewrite_rules", feed.UrlRewriteRules),
			slog.Any("error", err))
		return
	}

	rewrittenURL := re.ReplaceAllString(entry.URL, parts[2])
	log.Debug("Rewriting entry URL",
		slog.String("original_entry_url", entry.URL),
		slog.String("rewritten_entry_url", rewrittenURL),
		slog.Int64("feed_id", feed.ID),
		slog.String("feed_url", feed.FeedURL))
	entry.URL = rewrittenURL
}

func scrape(ctx context.Context, feed *model.Feed, entry *model.Entry,
) (string, string, error) {
	startTime := time.Now()
	builder := fetcher.NewRequestBuilder().
		WithContext(ctx).
		WithUserAgent(feed.UserAgent, config.Opts.HTTPClientUserAgent()).
		WithCookie(feed.Cookie).
		WithTimeout(config.Opts.HTTPClientTimeout()).
		WithProxyRotator(proxyrotator.ProxyRotatorInstance).
		WithCustomFeedProxyURL(feed.ProxyURL).
		WithCustomApplicationProxyURL(config.Opts.HTTPClientProxyURL()).
		UseCustomApplicationProxyURL(feed.FetchViaProxy).
		IgnoreTLSErrors(feed.AllowSelfSignedCertificates).
		DisableHTTP2(feed.DisableHTTP2)

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
	rewriteEntryURL(ctx, feed, entry)
	baseURL, content, err := scrape(ctx, feed, entry)
	if err != nil || content == "" {
		return err
	}

	if user.ShowReadingTime {
		entry.ReadingTime = readingtime.EstimateReadingTime(entry.Content,
			user.DefaultReadingSpeed, user.CJKReadingSpeed)
	}

	rewrite.Rewriter(entry.URL, entry, entry.Feed.RewriteRules)
	entry.Content = sanitizer.SanitizeHTML(baseURL, entry.Content,
		&sanitizer.SanitizerOptions{
			OpenLinksInNewTab: !user.OpenExternalLinkSameTab(),
		})
	return nil
}
