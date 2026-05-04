package template

import (
	"iter"
	"strings"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/sanitizer"
)

type Entry struct {
	*model.Entry

	urlSafeOnce memorized[bool]
}

func NewEntry(entry *model.Entry) *Entry { return &Entry{Entry: entry} }

func Entries(entries model.Entries) iter.Seq[*Entry] {
	return func(yield func(*Entry) bool) {
		for _, entry := range entries {
			wrappedEntry := Entry{Entry: entry}
			if !yield(&wrappedEntry) {
				return
			}
		}
	}
}

func (self *Entry) URLSafe() bool {
	return self.urlSafeOnce.From(self.urlSafeDo)
}

func (self *Entry) urlSafeDo() bool {
	protocol, _, ok := strings.Cut(self.URL, ":")
	if ok && !strings.Contains(protocol, "/") {
		for _, safeScheme := range [...]string{"http", "https", "mailto"} {
			if strings.EqualFold(protocol, safeScheme) {
				return true
			}
		}
	}

	u, err := self.ParsedURL()
	if err != nil {
		return false
	}
	return sanitizer.AllowedURLScheme(u)
}
