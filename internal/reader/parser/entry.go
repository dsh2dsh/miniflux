package parser

import (
	"time"

	"miniflux.app/v2/internal/model"
)

func NewEntry(feed *model.Feed) *model.Entry {
	return &model.Entry{
		Date:   time.Now(),
		Feed:   feed,
		Status: model.EntryStatusUnread,
	}
}
