package filter

import (
	"log/slog"

	"miniflux.app/v2/internal/model"
)

func makeUniqEntries(feed *model.Feed) uniqEntries {
	return uniqEntries{
		feed: feed,
		seen: make(map[string]*model.Entry, len(feed.Entries)),
	}
}

type uniqEntries struct {
	feed *model.Feed
	seen map[string]*model.Entry
}

func (self *uniqEntries) Add(e *model.Entry, log *slog.Logger) bool {
	prev, seen := self.seen[e.Hash]
	if !seen {
		self.seen[e.Hash] = e
		return true
	}

	self.feed.IncRemovedBroken()
	log.Info(
		"reader/filter: Block broken entry with the same hash",
		slog.Group("entry",
			slog.String("hash", e.Hash),
			slog.String("url", e.URL),
			slog.String("title", e.Title)),
		slog.Group("prev_entry",
			slog.String("url", prev.URL),
			slog.String("title", prev.Title)))
	return false
}
