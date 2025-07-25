// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package icon // import "miniflux.app/v2/internal/reader/icon"

import (
	"context"
	"log/slog"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/storage"
)

type IconChecker struct {
	store *storage.Storage
	feed  *model.Feed
}

func NewIconChecker(store *storage.Storage, feed *model.Feed) *IconChecker {
	return &IconChecker{
		store: store,
		feed:  feed,
	}
}

func (c *IconChecker) fetchAndStoreIcon(ctx context.Context) {
	requestBuilder := fetcher.NewRequestFeed(c.feed)

	iconFinder := NewIconFinder(requestBuilder, c.feed.SiteURL, c.feed.IconURL,
		config.Opts.PreferSiteIcon())
	if icon, err := iconFinder.FindIcon(); err != nil {
		slog.Debug("Unable to find feed icon",
			slog.Int64("feed_id", c.feed.ID),
			slog.String("website_url", c.feed.SiteURL),
			slog.String("feed_icon_url", c.feed.IconURL),
			slog.Any("error", err),
		)
	} else if icon == nil {
		slog.Debug("No icon found",
			slog.Int64("feed_id", c.feed.ID),
			slog.String("website_url", c.feed.SiteURL),
			slog.String("feed_icon_url", c.feed.IconURL),
		)
	} else {
		if err := c.store.StoreFeedIcon(ctx, c.feed.ID, icon); err != nil {
			slog.Error("Unable to store feed icon",
				slog.Int64("feed_id", c.feed.ID),
				slog.String("website_url", c.feed.SiteURL),
				slog.String("feed_icon_url", c.feed.IconURL),
				slog.Any("error", err),
			)
		} else {
			slog.Debug("Feed icon stored",
				slog.Int64("feed_id", c.feed.ID),
				slog.String("website_url", c.feed.SiteURL),
				slog.String("feed_icon_url", c.feed.IconURL),
				slog.Int64("icon_id", icon.ID),
				slog.String("icon_hash", icon.Hash),
			)
		}
	}
}

func (c *IconChecker) CreateFeedIconIfMissing(ctx context.Context) {
	if c.store.HasFeedIcon(ctx, c.feed.ID) {
		slog.Debug("Feed icon already exists",
			slog.Int64("feed_id", c.feed.ID),
		)
		return
	}
	c.fetchAndStoreIcon(ctx)
}

func (c *IconChecker) UpdateOrCreateFeedIcon(ctx context.Context) {
	c.fetchAndStoreIcon(ctx)
}
