package parser

import (
	"bytes"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dsh2dsh/gofeed/v2/atom"
	"github.com/dsh2dsh/gofeed/v2/options"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"
)

type atomFeed struct {
	baseURL *url.URL
	atom    *atom.Feed
	feed    *model.Feed

	parsedSiteURL *url.URL
}

func parseAtom(feedURL *url.URL, b []byte) (*model.Feed, error) {
	parsed, err := atom.NewParser().Parse(bytes.NewReader(b),
		options.WithSkipUnknownElements(true))
	if err != nil {
		return nil, fmt.Errorf("reader/parser: parse Atom feed: %w", err)
	}

	var p atomFeed
	return p.Feed(feedURL, parsed)
}

func (self *atomFeed) Feed(feedURL *url.URL, atomFeed *atom.Feed,
) (*model.Feed, error) {
	self.baseURL, self.atom = feedURL, atomFeed

	self.feed = &model.Feed{
		Title:       self.atom.Title,
		FeedURL:     self.feedURL(),
		SiteURL:     self.siteURL(),
		Description: self.atom.Subtitle,
		IconURL:     self.iconURL(),
	}
	self.feed.Entries = self.entries()
	return self.feed, nil
}

func (self *atomFeed) feedURL() string {
	link := self.atom.GetFeedLink()
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

func (self *atomFeed) siteURL() string {
	link := self.atom.GetLink()
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

func (self *atomFeed) iconURL() string {
	imageURL := self.atom.ImageURL()
	if imageURL == "" {
		return ""
	}

	u, err := url.Parse(imageURL)
	if err != nil || u.IsAbs() || self.parsedSiteURL == nil {
		return imageURL
	}
	return self.parsedSiteURL.ResolveReference(u).String()
}

func (self *atomFeed) entries() model.Entries {
	if len(self.atom.Entries) == 0 {
		return nil
	}

	entries := make(model.Entries, len(self.atom.Entries))
	for i, item := range self.atom.Entries {
		entries[i] = self.entry(item)
	}
	return entries
}

func (self *atomFeed) entry(item *atom.Entry) *model.Entry {
	p := atomEntry{
		atom:    item,
		siteURL: self.parsedSiteURL,
		entry:   model.NewEntry(),
	}

	entry := p.Parse()
	entry.Author = self.entryAuthor(entry)
	entry.Tags = self.entryTags(entry)
	entry.URL = self.entryURL(entry)
	entry.Title = self.entryTitle(entry)
	return entry
}

func (self *atomFeed) entryAuthor(entry *model.Entry) string {
	if entry.Author != "" {
		return entry.Author
	}
	return joinAtomAuthors(self.atom.Authors)
}

func joinAtomAuthors(authors []*atom.Person) string {
	switch len(authors) {
	case 0:
		return ""
	case 1:
		a := authors[0]
		if a.Name != "" {
			return a.Name
		}
		return a.Email
	}

	names := make([]string, len(authors))
	for i, a := range authors {
		if a.Name != "" {
			names[i] = a.Name
		} else {
			names[i] = a.Email
		}
	}

	slices.Sort(names)
	names = slices.Compact(names)
	return strings.Join(names, ", ")
}

func (self *atomFeed) entryTags(entry *model.Entry) []string {
	tags := entry.Tags
	if len(tags) == 0 {
		tags = self.atom.GetCategories()
	}

	if len(tags) < 2 {
		return tags
	}

	slices.Sort(tags)
	return slices.Compact(tags)
}

func (self *atomFeed) entryURL(entry *model.Entry) string {
	if entry.URL != "" {
		return entry.URL
	}
	return self.feed.SiteURL
}

func (self *atomFeed) entryTitle(entry *model.Entry) string {
	if entry.Title == "" && entry.Content == "" {
		return entry.URL
	}
	return entry.Title
}

type atomEntry struct {
	atom    *atom.Entry
	siteURL *url.URL

	entry *model.Entry
}

func (self *atomEntry) Parse() *model.Entry {
	self.entry.Date = self.published()
	self.entry.Title = self.atom.Title
	self.entry.Content = self.atom.GetContent()
	self.entry.URL = self.itemURL()
	self.entry.Author = joinAtomAuthors(self.atom.Authors)
	self.entry.CommentsURL = self.commentsURL()
	self.entry.Tags = self.atom.GetCategories()
	self.entry.Hash = self.hash()
	self.entry.AppendEnclosures(self.enclosures())

	enclosures := self.entry.Enclosures()
	if len(enclosures) != 0 && self.entry.URL == "" {
		self.entry.URL = enclosures[0].URL
	}
	return self.entry
}

func (self *atomEntry) published() time.Time {
	if t := self.atom.GetPublishedParsed(); t != nil {
		return *t
	}
	return time.Now()
}

func (self *atomEntry) itemURL() string {
	link := self.atom.GetLink()
	u, err := url.Parse(link)
	if err != nil {
		return ""
	} else if u.IsAbs() || self.siteURL == nil {
		return link
	}
	return self.siteURL.ResolveReference(u).String()
}

func (self *atomEntry) commentsURL() string {
	contentTypes := [...]string{"text/html", "application/xhtml+xml"}
	for _, link := range self.atom.Links {
		if !strings.EqualFold(link.Rel, "replies") {
			continue
		}
		for _, contentType := range contentTypes {
			if !strings.EqualFold(link.Type, contentType) {
				continue
			}
			u, err := url.Parse(link.Href)
			switch {
			case err != nil:
				continue
			case u.IsAbs():
				return link.Href
			case self.siteURL == nil:
				continue
			}
			return self.siteURL.ResolveReference(u).String()
		}
	}
	return ""
}

func (self *atomEntry) hash() string {
	switch {
	case self.entry.URL != "":
		return crypto.HashFromString(self.entry.URL)
	case self.atom.ID != "":
		return crypto.HashFromString(self.atom.ID)
	default:
		return crypto.HashFromString(self.entry.Title + self.entry.Content)
	}
}

func (self *atomEntry) enclosures() []model.Enclosure {
	enclosures := slices.Collect(self.atomEnclosures())
	if self.atom.Media != nil {
		enclosures = slices.AppendSeq(enclosures, self.mediaThumbnails())
		enclosures = slices.AppendSeq(enclosures, self.mediaContents())
		enclosures = slices.AppendSeq(enclosures, self.mediaPeerLinks())
	}

	for i := range enclosures {
		enc := &enclosures[i]
		if u, err := url.Parse(enc.URL); err != nil {
			enc.URL = ""
		} else if !u.IsAbs() {
			if self.siteURL == nil {
				enc.URL = ""
			} else {
				enc.URL = self.siteURL.ResolveReference(u).String()
			}
		}
	}

	enclosures = slices.DeleteFunc(enclosures, func(enc model.Enclosure) bool {
		return enc.URL == ""
	})
	return enclosures
}

func (self *atomEntry) atomEnclosures() iter.Seq[model.Enclosure] {
	return func(yield func(model.Enclosure) bool) {
		if len(self.atom.Links) == 0 {
			return
		}

		for _, link := range self.atom.Links {
			if link.Rel != "enclosure" || link.Href == "" {
				continue
			}

			enc := model.Enclosure{URL: link.Href, MimeType: link.Type}
			if s := link.Length; s != "" {
				size, err := strconv.ParseInt(s, 10, 64)
				if err == nil {
					enc.Size = size
				}
			}

			if !yield(enc) {
				return
			}
		}
	}
}

func (self *atomEntry) mediaThumbnails() iter.Seq[model.Enclosure] {
	return func(yield func(model.Enclosure) bool) {
		for thumbnail := range self.atom.Media.AllThumbnails() {
			enc := model.Enclosure{URL: thumbnail, MimeType: "image/*"}
			if !yield(enc) {
				return
			}
		}
	}
}

func (self *atomEntry) mediaContents() iter.Seq[model.Enclosure] {
	return func(yield func(model.Enclosure) bool) {
		for content := range self.atom.Media.AllContents() {
			enc := model.Enclosure{URL: content.URL, MimeType: content.Type}
			if enc.MimeType == "" {
				switch content.Medium {
				case "image":
					enc.MimeType = "image/*"
				case "video":
					enc.MimeType = "video/*"
				case "audio":
					enc.MimeType = "audio/*"
				default:
					enc.MimeType = "application/octet-stream"
				}
			}

			if s := content.FileSize; s != "" {
				size, err := strconv.ParseInt(content.FileSize, 10, 64)
				if err == nil {
					enc.Size = size
				}
			}

			if !yield(enc) {
				return
			}
		}
	}
}

func (self *atomEntry) mediaPeerLinks() iter.Seq[model.Enclosure] {
	return func(yield func(model.Enclosure) bool) {
		for pl := range self.atom.Media.AllPeerLinks() {
			enc := model.Enclosure{URL: pl.URL, MimeType: pl.Type}
			if enc.MimeType == "" {
				enc.MimeType = "application/octet-stream"
			}
			if !yield(enc) {
				return
			}
		}
	}
}
