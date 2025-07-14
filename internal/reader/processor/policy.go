package processor

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"miniflux.app/v2/internal/model"
)

var titlePolicy = bluemonday.StrictPolicy()

func sanitizeTitle(entry *model.Entry) {
	entry.Title = strings.TrimSpace(entry.Title)
	if entry.Title == "" {
		return
	}
	entry.Title = titlePolicy.Sanitize(entry.Title)
}
