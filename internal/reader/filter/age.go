package filter

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

func DeleteAgedEntries(ctx context.Context, feed *model.Feed) {
	maxAge := config.FilterEntryMaxAge()
	if maxAge == 0 {
		return
	}

	feed.Entries = slices.DeleteFunc(feed.Entries, func(e *model.Entry) bool {
		if entryAged(ctx, e, maxAge) {
			feed.IncFilteredByAge()
			return true
		}
		return false
	})
}

func entryAged(ctx context.Context, entry *model.Entry, maxAge time.Duration,
) bool {
	if maxAge == 0 {
		return false
	}

	if entry.Date.Add(maxAge).Before(time.Now()) {
		logging.FromContext(ctx).Debug("Entry is blocked globally due to max age",
			slog.String("entry_url", entry.URL),
			slog.Time("entry_date", entry.Date),
			slog.Duration("max_age", maxAge))
		return true
	}
	return false
}
