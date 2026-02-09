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

	"miniflux.app/v2/internal/model"
)

type atomFeed struct {
	baseURL *url.URL
	atom    *atom.Feed
	feed    *model.Feed
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
		Description: self.atom.Subtitle,
	}

	self.feed.WithFeedURL(self.feedURL())
	self.feed.WithSiteURL(self.siteURL())
	self.feed.IconURL = self.iconURL()
	self.feed.Entries = self.entries()
	return self.feed, nil
}

func (self *atomFeed) feedURL() *url.URL {
	link := self.atom.GetFeedLink()
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

func (self *atomFeed) siteURL() *url.URL {
	link := self.atom.GetLink()
	u, err := url.Parse(link)
	if err != nil {
		return self.baseURL
	} else if u.IsAbs() {
		return u
	}
	return self.baseURL.ResolveReference(u)
}

func (self *atomFeed) iconURL() string {
	imageURL := self.atom.ImageURL()
	if imageURL == "" {
		return ""
	}

	u, err := url.Parse(imageURL)
	if err != nil || u.IsAbs() {
		return imageURL
	}
	return self.ResolveReference(u).String()
}

func (self *atomFeed) ResolveReference(u *url.URL) *url.URL {
	siteURL, err := self.feed.ParsedSiteURL()
	if err != nil || siteURL == nil {
		return u
	}
	return siteURL.ResolveReference(u)
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
	p := atomEntry{feed: self, atom: item, entry: NewEntry(self.feed)}
	entry := p.Parse()
	self.fixEntryURL(entry)
	entry.Author = self.entryAuthor(entry)
	entry.Tags = self.entryTags(entry)
	return entry.WithAtom(item)
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

func (self *atomFeed) fixEntryURL(entry *model.Entry) {
	if entry.URL != "" {
		return
	}

	if u, err := self.feed.ParsedSiteURL(); err == nil {
		entry.WithURL(u)
	} else {
		entry.WithURLString(self.feed.SiteURL)
	}
}

type atomEntry struct {
	feed  *atomFeed
	atom  *atom.Entry
	entry *model.Entry
}

func (self *atomEntry) Parse() *model.Entry {
	self.entry.Date = self.published()
	self.entry.Title = self.atom.Title
	self.entry.Content = self.atom.GetContent()
	self.entry.WithURL(self.itemURL())
	self.entry.Author = joinAtomAuthors(self.atom.Authors)
	self.entry.CommentsURL = self.commentsURL()
	self.entry.Tags = self.atom.GetCategories()
	self.entry.AppendEnclosures(self.enclosures())
	self.hashEntry()

	enclosures := self.entry.Enclosures()
	if len(enclosures) != 0 && self.entry.URL == "" {
		u, _ := enclosures[0].ParsedURL()
		self.entry.WithURL(u)
	}
	return self.entry
}

func (self *atomEntry) published() time.Time {
	if t := self.atom.GetPublishedParsed(); t != nil {
		return *t
	}
	return time.Now()
}

func (self *atomEntry) itemURL() *url.URL {
	link := self.atom.GetLink()
	u, err := url.Parse(link)
	if err != nil {
		return nil
	} else if u.IsAbs() {
		return u
	}
	return self.feed.ResolveReference(u)
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
			}
			return self.feed.ResolveReference(u).String()
		}
	}
	return ""
}

func (self *atomEntry) hashEntry() {
	switch {
	case self.entry.URL != "":
		self.entry.HashFrom(self.entry.URL)
	case self.atom.ID != "":
		self.entry.HashFrom(self.atom.ID)
	default:
		self.entry.HashFrom(self.entry.Title + self.entry.Content)
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
		if u, err := enc.ParsedURL(); err != nil {
			enc.URL = ""
		} else if !u.IsAbs() {
			enc.WithURL(self.feed.ResolveReference(u))
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
		for t := range self.atom.Media.AllThumbnailsEx() {
			enc := model.Enclosure{
				URL:      t.URL,
				MimeType: "image/*",
				Height:   t.Height,
				Width:    t.Width,
			}
			if !yield(enc) {
				return
			}
		}
	}
}

func (self *atomEntry) mediaContents() iter.Seq[model.Enclosure] {
	return func(yield func(model.Enclosure) bool) {
		for content := range self.atom.Media.AllContents() {
			enc := model.Enclosure{
				URL:      content.URL,
				MimeType: content.Type,
				Height:   content.Height,
				Width:    content.Width,
			}
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
