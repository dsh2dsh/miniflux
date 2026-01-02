package filter

import (
	"log/slog"
	"slices"

	"miniflux.app/v2/internal/model"
)

type authors struct {
	block  []string
	logger *slog.Logger
}

func NewAuthors(block []string) *authors {
	self := &authors{block: block}
	return self.init()
}

func (self *authors) init() *authors {
	if !slices.IsSorted(self.block) {
		slices.Sort(self.block)
	}
	return self
}

func (self *authors) WithLogger(l *slog.Logger) *authors {
	self.logger = l
	return self
}

func (self *authors) Match(entry *model.Entry) bool {
	if entry.Author == "" || len(self.block) == 0 {
		return false
	}

	_, found := slices.BinarySearch(self.block, entry.Author)
	if found {
		self.logMatch(entry)
	}
	return found
}

func (self *authors) logMatch(entry *model.Entry) {
	if self.logger == nil {
		return
	}
	self.logger.Debug("Filtering entry blocked by author",
		slog.String("author", entry.Author),
		slog.String("entry_url", entry.URL))
}
