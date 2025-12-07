package parser

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dsh2dsh/gofeed/v2/options"
	"github.com/dsh2dsh/gofeed/v2/rss"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"
)

type rssFeed struct {
	baseURL *url.URL
	rss     *rss.Feed
	feed    *model.Feed

	parsedSiteURL *url.URL
}

func parseRSS(feedURL *url.URL, b []byte) (*model.Feed, error) {
	parsed, err := rss.NewParser().Parse(bytes.NewReader(b),
		options.WithSkipUnknownElements(true))
	if err != nil {
		return nil, fmt.Errorf("reader/parser: parse RSS feed: %w", err)
	}

	var p rssFeed
	return p.Feed(feedURL, parsed)
}

func (self *rssFeed) Feed(feedURL *url.URL, rssFeed *rss.Feed,
) (*model.Feed, error) {
	self.baseURL, self.rss = feedURL, rssFeed

	self.feed = &model.Feed{
		Title:       self.rss.GetTitle(),
		FeedURL:     self.feedURL(),
		SiteURL:     self.siteURL(),
		Description: self.rss.GetDescription(),
		TTL:         self.rss.GetTTL(),
		IconURL:     self.iconURL(),
	}
	self.feed.Entries = self.entries()
	return self.feed, nil
}

func (self *rssFeed) feedURL() string {
	link := self.rss.FeedLink()
	if link == "" {
		return self.baseURL.String()
	}

	u, err := url.Parse(link)
	if err != nil {
		return self.baseURL.String()
	} else if u.IsAbs() {
		return link
	}
	return self.baseURL.ResolveReference(u).String()
}

func (self *rssFeed) siteURL() string {
	link := self.rss.Link()
	u, err := url.Parse(link)
	if err != nil {
		return link
	} else if u.IsAbs() {
		self.parsedSiteURL = u
		return link
	}

	self.parsedSiteURL = self.baseURL.ResolveReference(u)
	return self.parsedSiteURL.String()
}

func (self *rssFeed) iconURL() string {
	img := self.rss.GetImage()
	if img == nil {
		return ""
	}

	u, err := url.Parse(img.URL)
	if err != nil || u.IsAbs() || self.parsedSiteURL == nil {
		return img.URL
	}
	return self.parsedSiteURL.ResolveReference(u).String()
}

func (self *rssFeed) entries() []*model.Entry {
	if len(self.rss.Items) == 0 {
		return nil
	}

	entries := make(model.Entries, len(self.rss.Items))
	for i, item := range self.rss.Items {
		entries[i] = self.entry(item)
	}
	return entries
}

func (self *rssFeed) entry(item *rss.Item) *model.Entry {
	p := rssEntry{
		rss:     item,
		siteURL: self.parsedSiteURL,
		entry:   &model.Entry{Date: time.Now(), Feed: self.feed},
	}

	entry := p.Parse()
	entry.Author = self.entryAuthor(entry)
	entry.Tags = self.entryTags(entry)
	entry.URL = self.entryURL(entry)
	entry.Title = self.entryTitle(entry)
	return entry
}

func (self *rssFeed) entryAuthor(entry *model.Entry) string {
	if entry.Author != "" {
		return entry.Author
	}

	name, address, ok := self.rss.GetAuthor()
	switch {
	case !ok:
		return ""
	case name != "":
		return name
	case address != "":
		return address
	}
	return ""
}

func (self *rssFeed) entryTags(entry *model.Entry) []string {
	tags := entry.Tags
	if len(tags) == 0 {
		tags = slices.Collect(self.rss.AllCategories())
	}

	if len(tags) < 2 {
		return tags
	}

	slices.Sort(tags)
	return slices.Compact(tags)
}

func (self *rssFeed) entryURL(entry *model.Entry) string {
	if entry.URL != "" {
		return entry.URL
	}
	return self.feed.SiteURL
}

func (self *rssFeed) entryTitle(entry *model.Entry) string {
	if entry.Title == "" && entry.Content == "" {
		return entry.URL
	}
	return entry.Title
}

type rssEntry struct {
	rss     *rss.Item
	siteURL *url.URL

	entry *model.Entry
}

func (self *rssEntry) Parse() *model.Entry {
	self.entry.Date = self.published()
	self.entry.Title = self.rss.GetTitle()
	self.entry.Content = self.rss.GetContent()
	self.entry.URL = self.entryURL()
	self.entry.Author = self.author()
	self.entry.CommentsURL = self.commentsURL()
	self.entry.ReadingTime = self.readingTime()
	self.entry.Tags = slices.Collect(self.rss.AllCategories())
	self.entry.Hash = self.hash()
	self.entry.AppendEnclosures(self.enclosures())

	enclosures := self.entry.Enclosures()
	if len(enclosures) != 0 && self.entry.URL == "" {
		self.entry.URL = enclosures[0].URL
	}
	return self.entry
}

func (self *rssEntry) published() time.Time {
	if t := self.rss.GetPublishedParsed(); t != nil {
		return *t
	}
	return time.Now()
}

func (self *rssEntry) entryURL() string {
	link := self.rss.Link()
	if link == "" {
		return link
	}

	u, err := url.Parse(link)
	if err != nil {
		return ""
	} else if u.IsAbs() || self.siteURL == nil {
		return link
	}
	return self.siteURL.ResolveReference(u).String()
}

func (self *rssEntry) author() string {
	name, address, ok := self.rss.GetAuthor()
	switch {
	case !ok:
		return ""
	case name != "":
		return name
	case address != "":
		return address
	}
	return ""
}

func (self *rssEntry) commentsURL() string {
	if self.rss.Comments == "" {
		return ""
	}

	u, err := url.Parse(self.rss.Comments)
	switch {
	case err != nil:
		return ""
	case u.IsAbs():
		return self.rss.Comments
	case self.siteURL == nil:
		return ""
	}
	return self.siteURL.ResolveReference(u).String()
}

func (self *rssEntry) readingTime() int {
	if self.rss.ITunesExt == nil || self.rss.ITunesExt.Duration == "" {
		return 0
	}
	duration := self.rss.ITunesExt.Duration

	n := strings.Count(duration, ":")
	if n > 2 {
		return 0
	}

	var i, seconds int
	for s := range strings.SplitSeq(duration, ":") {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		seconds += int(math.Pow(60, float64(n-i))) * v
		i++
	}
	return seconds / 60
}

func (self *rssEntry) enclosures() (enclosures []model.Enclosure) {
	for rssEnc := range self.rss.AllEnclosures() {
		enc := model.Enclosure{URL: rssEnc.URL, MimeType: rssEnc.Type}
		if u, err := url.Parse(enc.URL); err != nil {
			continue
		} else if !u.IsAbs() {
			if self.siteURL == nil {
				continue
			} else {
				enc.URL = self.siteURL.ResolveReference(u).String()
			}
		}

		if s := rssEnc.Length; s != "" {
			if size, err := strconv.ParseInt(s, 10, 64); err == nil {
				enc.Size = size
			}
		}
		enclosures = append(enclosures, enc)
	}
	return enclosures
}

func (self *rssEntry) hash() string {
	switch {
	case self.entry.URL != "":
		return crypto.HashFromString(self.entry.URL)
	case self.rss.GUID != nil:
		return crypto.HashFromString(self.rss.GUID.Value)
	}
	return crypto.HashFromString(self.entry.Title + self.entry.Content)
}
