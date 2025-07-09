package storage

import (
	"context"
	"sync"

	"miniflux.app/v2/internal/model"
)

type DedupEntries struct {
	mu     sync.Mutex
	hashes map[int64]map[string]int64
	dedups int
}

type ctxDedupEntries struct{}

var DedupEntriesKey ctxDedupEntries = struct{}{}

func WithDedupEntries(ctx context.Context) (context.Context, *DedupEntries) {
	dd := NewDedupEntries()
	return context.WithValue(ctx, DedupEntriesKey, dd), dd
}

func NewDedupEntries() *DedupEntries {
	return &DedupEntries{hashes: make(map[int64]map[string]int64)}
}

func DedupEntriesFrom(ctx context.Context) *DedupEntries {
	if s, ok := ctx.Value(DedupEntriesKey).(*DedupEntries); ok {
		return s
	}
	return nil
}

func (self *DedupEntries) Dedups() int { return self.dedups }

func (self *DedupEntries) Filter(userID int64, entries model.Entries) int {
	if len(entries) == 0 {
		return 0
	}
	self.mu.Lock()
	defer self.mu.Unlock()

	hashes, found := self.hashes[userID]
	if !found {
		hashes = make(map[string]int64)
	}

	var dedups int
	for _, e := range entries.Unread() {
		if feedID, ok := hashes[e.Hash]; ok && e.FeedID != feedID {
			e.Status = model.EntryStatusRead
			dedups++
		} else if !ok {
			hashes[e.Hash] = e.FeedID
		}
	}

	if !found && len(hashes) != 0 {
		self.hashes[userID] = hashes
	}
	self.dedups += dedups
	return dedups
}
