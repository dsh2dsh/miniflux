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

	parsedSiteURL *url.URL
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
		FeedURL:     self.feedURL(),
		SiteURL:     self.siteURL(),
		Description: self.json.Description,
		IconURL:     self.iconURL(),
	}
	self.feed.Entries = self.entries()
	return self.feed, nil
}

func (self *jsonFeed) feedURL() string {
	link := strings.TrimSpace(self.json.FeedURL)
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

func (self *jsonFeed) siteURL() string {
	link := strings.TrimSpace(self.json.HomePageURL)
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

func (self *jsonFeed) iconURL() string {
	for _, link := range [...]string{self.json.Favicon, self.json.Icon} {
		if link = strings.TrimSpace(link); link == "" {
			continue
		}

		u, err := url.Parse(link)
		if err != nil {
			continue
		} else if u.IsAbs() || self.parsedSiteURL == nil {
			return link
		}
		return self.parsedSiteURL.ResolveReference(u).String()
	}
	return ""
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
	p := jsonEntry{
		json:    item,
		siteURL: self.parsedSiteURL,
		entry:   &model.Entry{Date: time.Now(), Feed: self.feed},
	}

	entry := p.Parse()
	entry.Author = self.entryAuthor(entry)
	entry.URL = self.entryURL(entry)
	entry.Title = self.entryTitle(entry)
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

func (self *jsonFeed) entryURL(entry *model.Entry) string {
	if entry.URL != "" {
		return entry.URL
	}
	return self.feed.SiteURL
}

func (self *jsonFeed) entryTitle(entry *model.Entry) string {
	if entry.Title == "" && entry.Content == "" {
		return entry.URL
	}
	return entry.Title
}

type jsonEntry struct {
	json    *json.Item
	siteURL *url.URL

	entry *model.Entry
}

func (self *jsonEntry) Parse() *model.Entry {
	self.entry.Date = self.published()
	self.entry.Title = self.json.Title
	self.entry.Content = self.json.Content()
	self.entry.URL = self.entryURL()
	self.entry.Author = joinJsonAuthors(self.json.AllAuthors())
	self.entry.Tags = self.tags()
	self.entry.AppendEnclosures(self.enclosures())
	self.hashEntry()

	enclosures := self.entry.Enclosures()
	if len(enclosures) != 0 && self.entry.URL == "" {
		self.entry.URL = enclosures[0].URL
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

func (self *jsonEntry) entryURL() string {
	for link := range self.json.AllLinks() {
		u, err := url.Parse(link)
		if err != nil {
			continue
		} else if u.IsAbs() || self.siteURL == nil {
			return link
		}
		return self.siteURL.ResolveReference(u).String()
	}
	return ""
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

		if u, err := url.Parse(enc.URL); err != nil {
			continue
		} else if !u.IsAbs() {
			if self.siteURL == nil {
				continue
			}
			enc.URL = self.siteURL.ResolveReference(u).String()
		}
		enclosures = append(enclosures, enc)
	}
	return enclosures
}
