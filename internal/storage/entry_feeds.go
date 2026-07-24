package storage

import (
	"time"

	"miniflux.app/v2/internal/model"
)

type entryFeeds struct {
	category model.Category
	feed     model.Feed
	icon     model.FeedIcon

	categories map[int64]*model.Category
	feeds      map[int64]*model.Feed
	icons      map[int64]*model.FeedIcon
}

func (self *entryFeeds) Init() {
	self.feed.Category = &self.category
	self.feed.Icon = &self.icon

	self.categories = make(map[int64]*model.Category)
	self.feeds = make(map[int64]*model.Feed)
	self.icons = make(map[int64]*model.FeedIcon)
}

func (self *entryFeeds) NewEntry() *model.Entry {
	entry := &model.Entry{
		Date: time.Now(),
		Feed: &self.feed,
		Tags: []string{},
	}
	return entry
}

func (self *entryFeeds) ReuseFeed(entry *model.Entry) {
	id := entry.FeedID
	if feed, ok := self.feeds[id]; ok {
		entry.Feed = feed
		return
	}

	feed := new(model.Feed)
	*feed = self.feed
	self.reuseCategory(feed)
	self.reuseIcon(feed)

	self.feeds[id] = feed
	entry.Feed = feed
}

func (self *entryFeeds) reuseCategory(feed *model.Feed) {
	id := feed.Category.ID
	if category, ok := self.categories[id]; ok {
		feed.Category = category
		return
	}

	category := new(model.Category)
	*category = self.category
	self.categories[id] = category
	feed.Category = category
}

func (self *entryFeeds) reuseIcon(feed *model.Feed) {
	id := feed.Icon.IconID
	if icon, ok := self.icons[id]; ok {
		feed.Icon = icon
		return
	}

	icon := new(model.FeedIcon)
	*icon = self.icon
	self.icons[id] = icon
	feed.Icon = icon
}
