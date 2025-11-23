package storage

import (
	"sync"
	"sync/atomic"

	"miniflux.app/v2/internal/model"
)

type DedupEntries struct {
	mu     sync.Mutex
	hashes map[int64]*dedupUserEntries

	created uint64
}

func NewDedupEntries() *DedupEntries {
	return &DedupEntries{hashes: make(map[int64]*dedupUserEntries)}
}

func (self *DedupEntries) Created() uint64 { return self.created }

func (self *DedupEntries) Filter(userID int64, r *model.FeedRefreshed) {
	if r.CreatedLen() == 0 {
		return
	}
	created := self.userHashes(userID).Filter(r)
	atomic.AddUint64(&self.created, created)
}

func (self *DedupEntries) userHashes(userID int64) *dedupUserEntries {
	self.mu.Lock()
	defer self.mu.Unlock()

	hashes, ok := self.hashes[userID]
	if !ok {
		hashes = &dedupUserEntries{hashes: make(map[string]int64)}
		self.hashes[userID] = hashes
	}
	return hashes
}

type dedupUserEntries struct {
	mu     sync.Mutex
	hashes map[string]int64
}

func (self *dedupUserEntries) Filter(r *model.FeedRefreshed) uint64 {
	self.mu.Lock()
	defer self.mu.Unlock()

	var created uint64
	for _, e := range r.Created.Unread() {
		if feedID, ok := self.hashes[e.Hash]; ok && e.FeedID != feedID {
			e.Status = model.EntryStatusRead
			r.Dedups++
		} else if !ok {
			self.hashes[e.Hash] = e.FeedID
			created++
		}
	}
	return created
}
