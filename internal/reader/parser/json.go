package parser

import (
	"bytes"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dsh2dsh/gofeed/v2/json"

	"miniflux.app/v2/internal/model"
)

type jsonFeed struct {
	baseURL *url.URL
	json    *json.Feed
	feed    *model.Feed
}

func parseJSON(feedURL *url.URL, b []byte) (*model.Feed, error) {
	parsed, err := json.NewParser().Parse(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("reader/parser: parse JSON feed: %w", err)
	}

	var p jsonFeed
	return p.Feed(feedURL, parsed)
}

func (self *jsonFeed) Feed(feedURL *url.URL, jsonFeed *json.Feed,
) (*model.Feed, error) {
	self.baseURL, self.json = feedURL, jsonFeed

	self.feed = &model.Feed{
		Title:       self.json.Title,
		Description: self.json.Description,
	}

	self.feed.WithFeedURL(self.feedURL())
	self.feed.WithSiteURL(self.siteURL())
	self.feed.IconURL = self.iconURL()
	self.feed.Entries = self.entries()
	return self.feed, nil
}

func (self *jsonFeed) feedURL() *url.URL {
	link := strings.TrimSpace(self.json.FeedURL)
	if link == "" {
		return self.baseURL
	}

	u, err := url.Parse(link)
	if err != nil {
		return self.baseURL
	} else if u.IsAbs() {
		return u
	}
	return self.baseURL.ResolveReference(u)
}

func (self *jsonFeed) siteURL() *url.URL {
	link := strings.TrimSpace(self.json.HomePageURL)
	u, err := url.Parse(link)
	if err != nil {
		return self.baseURL
	} else if u.IsAbs() {
		return u
	}
	return self.baseURL.ResolveReference(u)
}

func (self *jsonFeed) iconURL() string {
	for _, link := range [...]string{self.json.Favicon, self.json.Icon} {
		if link = strings.TrimSpace(link); link == "" {
			continue
		}

		u, err := url.Parse(link)
		if err != nil {
			continue
		} else if u.IsAbs() {
			return u.String()
		}
		return self.ResolveReference(u).String()
	}
	return ""
}

func (self *jsonFeed) ResolveReference(u *url.URL) *url.URL {
	siteURL, err := self.feed.ParsedSiteURL()
	if err != nil || siteURL == nil {
		return u
	}
	return siteURL.ResolveReference(u)
}

func (self *jsonFeed) entries() []*model.Entry {
	if len(self.json.Items) == 0 {
		return nil
	}

	entries := make(model.Entries, len(self.json.Items))
	for i, item := range self.json.Items {
		entries[i] = self.entry(item)
	}
	return entries
}

func (self *jsonFeed) entry(item *json.Item) *model.Entry {
	p := jsonEntry{feed: self, json: item, entry: NewEntry(self.feed)}
	entry := p.Parse()
	self.fixEntryURL(entry)

	entry.Author = self.entryAuthor(entry)
	return entry
}

func (self *jsonFeed) entryAuthor(entry *model.Entry) string {
	if entry.Author != "" {
		return entry.Author
	}
	return joinJsonAuthors(self.json.AllAuthors())
}

func joinJsonAuthors(authors iter.Seq[*json.Author]) string {
	var names []string
	for author := range authors {
		if author.Name != "" {
			names = append(names, author.Name)
		}
	}

	slices.Sort(names)
	names = slices.Compact(names)
	return strings.Join(names, ", ")
}

func (self *jsonFeed) fixEntryURL(entry *model.Entry) {
	if entry.URL != "" {
		return
	}

	if u, err := self.feed.ParsedSiteURL(); err == nil {
		entry.WithURL(u)
	} else {
		entry.WithURLString(self.feed.SiteURL)
	}
}

type jsonEntry struct {
	feed  *jsonFeed
	json  *json.Item
	entry *model.Entry
}

func (self *jsonEntry) Parse() *model.Entry {
	self.entry.Date = self.published()
	self.entry.Title = self.title()
	self.entry.Content = self.json.Content()
	self.entry.WithURL(self.entryURL())
	self.entry.Author = joinJsonAuthors(self.json.AllAuthors())
	self.entry.Tags = self.tags()
	self.entry.AppendEnclosures(self.enclosures())
	self.hashEntry()

	enclosures := self.entry.Enclosures()
	if len(enclosures) != 0 && self.entry.URL == "" {
		u, _ := enclosures[0].ParsedURL()
		self.entry.WithURL(u)
	}
	return self.entry
}

func (self *jsonEntry) published() time.Time {
	if t := self.json.PublishedParsed(); t != nil {
		return *t
	}
	if t := self.json.UpdatedParsed(); t != nil {
		return *t
	}
	return time.Now()
}

func (self *jsonEntry) title() string {
	for _, s := range [...]string{
		self.json.Title,
		self.json.Summary,
		self.json.ContentText,
		self.json.ContentHTML,
	} {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}

func (self *jsonEntry) entryURL() *url.URL {
	for link := range self.json.AllLinks() {
		u, err := url.Parse(link)
		if err != nil {
			continue
		} else if u.IsAbs() {
			return u
		}
		return self.feed.ResolveReference(u)
	}
	return nil
}

func (self *jsonEntry) tags() []string {
	tags := self.json.Tags
	if len(tags) == 0 {
		return nil
	}

	for i, s := range tags {
		tags[i] = strings.TrimSpace(s)
	}

	tags = slices.DeleteFunc(tags, func(s string) bool { return s == "" })
	switch len(tags) {
	case 0:
		return nil
	case 1:
		return tags
	}

	slices.Sort(tags)
	return slices.Compact(tags)
}

func (self *jsonEntry) hashEntry() {
	switch {
	case self.entry.URL != "":
		self.entry.HashFrom(self.entry.URL)
	case self.json.ID != "":
		self.entry.HashFrom(self.json.ID)
	default:
		self.entry.HashFrom(self.entry.Title + self.entry.Content)
	}
}

func (self *jsonEntry) enclosures() []model.Enclosure {
	if atts := self.json.Attachments; atts == nil || len(*atts) == 0 {
		return nil
	}

	enclosures := make([]model.Enclosure, 0, len(*self.json.Attachments))
	for _, att := range *self.json.Attachments {
		if att.URL = strings.TrimSpace(att.URL); att.URL == "" {
			continue
		}

		enc := model.Enclosure{
			URL:      att.URL,
			MimeType: att.MimeType,
			Size:     att.SizeInBytes,
		}

		if u, err := enc.ParsedURL(); err != nil {
			continue
		} else if !u.IsAbs() {
			enc.WithURL(self.feed.ResolveReference(u))
		}
		enclosures = append(enclosures, enc)
	}
	return enclosures
}
