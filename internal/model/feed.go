// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cespare/xxhash/v2"

	"miniflux.app/v2/internal/config"
)

const (
	// Default settings for the feed query builder
	DefaultFeedSorting          = "parsing_error_count"
	DefaultFeedSortingDirection = "desc"
)

// Feed represents a feed in the application.
type Feed struct {
	ID                          int64       `json:"id" db:"id"`
	UserID                      int64       `json:"user_id" db:"user_id"`
	FeedURL                     string      `json:"feed_url" db:"feed_url"`
	SiteURL                     string      `json:"site_url" db:"site_url"`
	Title                       string      `json:"title" db:"title"`
	Description                 string      `json:"description" db:"description"`
	CheckedAt                   time.Time   `json:"checked_at" db:"checked_at"`
	NextCheckAt                 time.Time   `json:"next_check_at" db:"next_check_at"`
	EtagHeader                  string      `json:"etag_header" db:"etag_header"`
	LastModifiedHeader          string      `json:"last_modified_header" db:"last_modified_header"`
	ParsingErrorMsg             string      `json:"parsing_error_message" db:"parsing_error_message"`
	ParsingErrorCount           int         `json:"parsing_error_count" db:"parsing_error_count"`
	ScraperRules                string      `json:"scraper_rules" db:"scraper_rules"`
	RewriteRules                string      `json:"rewrite_rules" db:"rewrite_rules"`
	UrlRewriteRules             string      `json:"urlrewrite_rules" db:"urlrewrite_rules"`
	UserAgent                   string      `json:"user_agent" db:"user_agent"`
	Cookie                      string      `json:"cookie" db:"cookie"`
	Username                    string      `json:"username" db:"username"`
	Password                    string      `json:"password" db:"password"`
	Disabled                    bool        `json:"disabled" db:"disabled"`
	NoMediaPlayer               bool        `json:"no_media_player" db:"no_media_player"`
	IgnoreHTTPCache             bool        `json:"ignore_http_cache" db:"ignore_http_cache"`
	AllowSelfSignedCertificates bool        `json:"allow_self_signed_certificates" db:"allow_self_signed_certificates"`
	FetchViaProxy               bool        `json:"fetch_via_proxy" db:"fetch_via_proxy"`
	HideGlobally                bool        `json:"hide_globally" db:"hide_globally"`
	DisableHTTP2                bool        `json:"disable_http2" db:"disable_http2"`
	PushoverEnabled             bool        `json:"pushover_enabled" db:"pushover_enabled"`
	NtfyEnabled                 bool        `json:"ntfy_enabled" db:"ntfy_enabled"`
	Crawler                     bool        `json:"crawler" db:"crawler"`
	AppriseServiceURLs          string      `json:"apprise_service_urls" db:"apprise_service_urls"`
	WebhookURL                  string      `json:"webhook_url" db:"webhook_url"`
	NtfyPriority                int         `json:"ntfy_priority" db:"ntfy_priority"`
	NtfyTopic                   string      `json:"ntfy_topic" db:"ntfy_topic"`
	PushoverPriority            int         `json:"pushover_priority" db:"pushover_priority"`
	ProxyURL                    string      `json:"proxy_url" db:"proxy_url"`
	Extra                       FeedExtra   `json:"extra,omitzero" db:"extra"`
	Runtime                     FeedRuntime `json:"runtime,omitzero" db:"runtime"`

	// Non-persisted attributes
	Category *Category `json:"category,omitempty"`
	Icon     *FeedIcon `json:"icon"`
	Entries  Entries   `json:"entries,omitempty"`

	// Internal attributes (not exposed in the API and not persisted in the database)
	TTL                    int    `json:"-" db:"-"`
	IconURL                string `json:"-" db:"-"`
	UnreadCount            int    `json:"-" db:"-"`
	ReadCount              int    `json:"-" db:"-"`
	NumberOfVisibleEntries int    `json:"-" db:"-"`

	filteredByAge    int
	filteredByRules  int
	filteredByHash   int
	filteredByStored int

	feedURL *url.URL
	siteURL *url.URL
}

type FeedExtra struct {
	BlockAuthors        []string `json:"blockAuthors,omitempty"`
	BlockMarkRead       bool     `json:"blockMarkRead,omitempty"`
	CommentsURLTemplate string   `json:"comments_url_template,omitempty"`

	BlockFilterEntryRules string `json:"block_filter_entry_rules,omitempty"`
	KeepFilterEntryRules  string `json:"keep_filter_entry_rules,omitempty"`
}

type FeedRuntime struct {
	Size uint64 `json:"size,omitempty"`
	Hash uint64 `json:"hash,omitempty"`
}

type FeedCounters struct {
	ReadCounters   map[int64]int `json:"reads"`
	UnreadCounters map[int64]int `json:"unreads"`
}

func (self *Feed) String() string {
	return fmt.Sprintf("ID=%d, UserID=%d, FeedURL=%s, SiteURL=%s, Title=%s, Category={%s}",
		self.ID,
		self.UserID,
		self.FeedURL,
		self.SiteURL,
		self.Title,
		self.Category,
	)
}

func (self *Feed) WithFeedURL(u *url.URL) *Feed {
	if u != nil {
		self.feedURL, self.FeedURL = u, u.String()
	} else {
		self.WithFeedURLString("")
	}
	return self
}

func (self *Feed) WithFeedURLString(feedURL string) *Feed {
	if self.FeedURL != feedURL {
		self.feedURL, self.FeedURL = nil, feedURL
	}
	return self
}

func (self *Feed) ParsedFeedURL() (*url.URL, error) {
	if self.feedURL != nil {
		return self.feedURL, nil
	}

	u, err := url.Parse(self.FeedURL)
	if err != nil {
		return nil, fmt.Errorf("parse feed URL: %w", err)
	}
	self.feedURL = u
	return u, nil
}

func (self *Feed) WithSiteURL(u *url.URL) *Feed {
	if u != nil {
		self.siteURL, self.SiteURL = u, u.String()
	} else {
		self.WithSiteURLString("")
	}
	return self
}

func (self *Feed) WithSiteURLString(siteURL string) *Feed {
	if self.SiteURL != siteURL {
		self.siteURL, self.SiteURL = nil, siteURL
	}
	return self
}

func (self *Feed) ParsedSiteURL() (*url.URL, error) {
	if self.siteURL != nil {
		return self.siteURL, nil
	}

	u, err := url.Parse(self.SiteURL)
	if err != nil {
		return nil, fmt.Errorf("parse site URL: %w", err)
	}
	self.siteURL = u
	return u, nil
}

// WithCategoryID initializes the category attribute of the feed.
func (self *Feed) WithCategoryID(categoryID int64) {
	self.Category = &Category{ID: categoryID}
}

// WithTranslatedErrorMessage adds a new error message and increment the error counter.
func (self *Feed) WithTranslatedErrorMessage(message string) {
	self.ParsingErrorCount++
	self.ParsingErrorMsg = message
}

// ResetErrorCounter removes all previous errors.
func (self *Feed) ResetErrorCounter() {
	self.ParsingErrorCount, self.ParsingErrorMsg = 0, ""
}

// CheckedNow set attribute values when the feed is refreshed.
func (self *Feed) CheckedNow() {
	self.CheckedAt = time.Now()
	if self.SiteURL == "" {
		self.SiteURL = self.FeedURL
	}
}

// ScheduleNextCheck set "next_check_at" of a feed.
func (self *Feed) ScheduleNextCheck(refreshDelayInMinutes int) int {
	// Default to the global config Polling Frequency.
	//
	// Use the RSS TTL field, Retry-After, Cache-Control or Expires HTTP headers
	// if defined.
	intervalMinutes := max(config.SchedulerRoundRobinMinInterval(),
		refreshDelayInMinutes)

	// Limit the max interval value for misconfigured feeds.
	intervalMinutes = min(config.SchedulerRoundRobinMaxInterval(),
		intervalMinutes)

	self.NextCheckAt = time.Now().Add(time.Minute * time.Duration(intervalMinutes))
	return intervalMinutes
}

func (self *Feed) Size() uint64 { return self.Runtime.Size }
func (self *Feed) HashString() string {
	return strconv.FormatUint(self.Runtime.Hash, 16)
}

func (self *Feed) ContentChanged(body []byte) bool {
	oldSize, oldHash := self.Runtime.Size, self.Runtime.Hash
	self.Runtime.Size, self.Runtime.Hash = uint64(len(body)), xxhash.Sum64(body)
	return self.Runtime.Size != oldSize || self.Runtime.Hash != oldHash
}

func (self *Feed) CommentsURLTemplate() (*template.Template, error) {
	s := strings.TrimSpace(self.Extra.CommentsURLTemplate)
	if s == "" {
		return nil, nil
	}

	t, err := template.New("").Funcs(commentsURLTemplateFuncMap).Parse(s)
	if err != nil {
		return nil, fmt.Errorf(
			"model: parsing comments_url_template %q: %w", s, err)
	}
	return t, nil
}

var commentsURLTemplateFuncMap = template.FuncMap{
	"replace": func(s, from, to string) string {
		return strings.Replace(s, from, to, 1)
	},
}

func (self *Feed) BlockAuthors() []string { return self.Extra.BlockAuthors }

func (self *Feed) WithBlockAuthors(authors []string) *Feed {
	if len(authors) == 0 {
		self.Extra.BlockAuthors = nil
		return self
	}

	slices.Sort(authors)
	self.Extra.BlockAuthors = slices.Compact(authors)
	return self
}

func (self *Feed) BlockMarkRead() bool { return self.Extra.BlockMarkRead }

func (self *Feed) WithBlockMarkRead(value bool) *Feed {
	self.Extra.BlockMarkRead = value
	return self
}

func (self *Feed) BlockFilterEntryRules() string {
	return self.Extra.BlockFilterEntryRules
}

func (self *Feed) KeepFilterEntryRules() string {
	return self.Extra.KeepFilterEntryRules
}

func (self *Feed) IncFilteredByAge()  { self.filteredByAge++ }
func (self *Feed) FilteredByAge() int { return self.filteredByAge }

func (self *Feed) IncFilteredByRules()  { self.filteredByRules++ }
func (self *Feed) FilteredByRules() int { return self.filteredByRules }

func (self *Feed) IncFilteredByHash()  { self.filteredByHash++ }
func (self *Feed) FilteredByHash() int { return self.filteredByHash }

func (self *Feed) IncFilteredByStored()  { self.filteredByStored++ }
func (self *Feed) FilteredByStored() int { return self.filteredByStored }

// FeedCreationRequest represents the request to create a feed.
type FeedCreationRequest struct {
	FeedURL                     string   `json:"feed_url"`
	CategoryID                  int64    `json:"category_id"`
	UserAgent                   string   `json:"user_agent"`
	Cookie                      string   `json:"cookie"`
	Username                    string   `json:"username"`
	Password                    string   `json:"password"`
	Crawler                     bool     `json:"crawler"`
	Disabled                    bool     `json:"disabled"`
	NoMediaPlayer               bool     `json:"no_media_player"`
	IgnoreHTTPCache             bool     `json:"ignore_http_cache"`
	AllowSelfSignedCertificates bool     `json:"allow_self_signed_certificates"`
	FetchViaProxy               bool     `json:"fetch_via_proxy"`
	HideGlobally                bool     `json:"hide_globally"`
	DisableHTTP2                bool     `json:"disable_http2"`
	ScraperRules                string   `json:"scraper_rules"`
	RewriteRules                string   `json:"rewrite_rules"`
	BlockAuthors                []string `json:"blockAuthors,omitempty"`
	BlockFilterEntryRules       string   `json:"block_filter_entry_rules"`
	BlockMarkRead               bool     `json:"blockMarkRead,omitempty"`
	KeepFilterEntryRules        string   `json:"keep_filter_entry_rules"`
	UrlRewriteRules             string   `json:"urlrewrite_rules"`
	ProxyURL                    string   `json:"proxy_url"`
}

type FeedCreationRequestFromSubscriptionDiscovery struct {
	FeedCreationRequest

	Content      []byte
	ETag         string
	LastModified string
}

// FeedModificationRequest represents the request to update a feed.
type FeedModificationRequest struct {
	FeedURL                     *string   `json:"feed_url"`
	SiteURL                     *string   `json:"site_url"`
	Title                       *string   `json:"title"`
	Description                 *string   `json:"description"`
	ScraperRules                *string   `json:"scraper_rules"`
	RewriteRules                *string   `json:"rewrite_rules"`
	UrlRewriteRules             *string   `json:"urlrewrite_rules"`
	BlockAuthors                *[]string `json:"blockAuthors,omitempty"`
	BlockFilterEntryRules       *string   `json:"block_filter_entry_rules"`
	KeepFilterEntryRules        *string   `json:"keep_filter_entry_rules"`
	Crawler                     *bool     `json:"crawler"`
	UserAgent                   *string   `json:"user_agent"`
	Cookie                      *string   `json:"cookie"`
	Username                    *string   `json:"username"`
	Password                    *string   `json:"password"`
	CategoryID                  *int64    `json:"category_id"`
	Disabled                    *bool     `json:"disabled"`
	NoMediaPlayer               *bool     `json:"no_media_player"`
	IgnoreHTTPCache             *bool     `json:"ignore_http_cache"`
	AllowSelfSignedCertificates *bool     `json:"allow_self_signed_certificates"`
	FetchViaProxy               *bool     `json:"fetch_via_proxy"`
	HideGlobally                *bool     `json:"hide_globally"`
	DisableHTTP2                *bool     `json:"disable_http2"`
	ProxyURL                    *string   `json:"proxy_url"`
	CommentsURLTemplate         *string   `json:"comments_url_template,omitempty"`
}

// Patch updates a feed with modified values.
func (self *FeedModificationRequest) Patch(feed *Feed) {
	if self.FeedURL != nil && *self.FeedURL != "" {
		feed.WithFeedURLString(*self.FeedURL)
	}

	if self.SiteURL != nil && *self.SiteURL != "" {
		feed.WithSiteURLString(*self.SiteURL)
	}

	if self.Title != nil && *self.Title != "" {
		feed.Title = *self.Title
	}

	if self.Description != nil && *self.Description != "" {
		feed.Description = *self.Description
	}

	if self.ScraperRules != nil {
		feed.ScraperRules = *self.ScraperRules
	}

	if self.RewriteRules != nil {
		feed.RewriteRules = *self.RewriteRules
	}

	if self.UrlRewriteRules != nil {
		feed.UrlRewriteRules = *self.UrlRewriteRules
	}

	if self.BlockAuthors != nil {
		feed.WithBlockAuthors(*self.BlockAuthors)
	}

	if self.BlockFilterEntryRules != nil {
		feed.Extra.BlockFilterEntryRules = *self.BlockFilterEntryRules
	}

	if self.KeepFilterEntryRules != nil {
		feed.Extra.KeepFilterEntryRules = *self.KeepFilterEntryRules
	}

	if self.Crawler != nil {
		feed.Crawler = *self.Crawler
	}

	if self.UserAgent != nil {
		feed.UserAgent = *self.UserAgent
	}

	if self.Cookie != nil {
		feed.Cookie = *self.Cookie
	}

	if self.Username != nil {
		feed.Username = *self.Username
	}

	if self.Password != nil {
		feed.Password = *self.Password
	}

	if self.CategoryID != nil && *self.CategoryID > 0 {
		feed.Category.ID = *self.CategoryID
	}

	if self.Disabled != nil {
		feed.Disabled = *self.Disabled
	}

	if self.NoMediaPlayer != nil {
		feed.NoMediaPlayer = *self.NoMediaPlayer
	}

	if self.IgnoreHTTPCache != nil {
		feed.IgnoreHTTPCache = *self.IgnoreHTTPCache
	}

	if self.AllowSelfSignedCertificates != nil {
		feed.AllowSelfSignedCertificates = *self.AllowSelfSignedCertificates
	}

	if self.FetchViaProxy != nil {
		feed.FetchViaProxy = *self.FetchViaProxy
	}

	if self.HideGlobally != nil {
		feed.HideGlobally = *self.HideGlobally
	}

	if self.DisableHTTP2 != nil {
		feed.DisableHTTP2 = *self.DisableHTTP2
	}

	if self.ProxyURL != nil {
		feed.ProxyURL = *self.ProxyURL
	}

	if self.CommentsURLTemplate != nil {
		feed.Extra.CommentsURLTemplate = *self.CommentsURLTemplate
	}
}

// Feeds is a list of feed
type Feeds []*Feed

type FeedRefreshed struct {
	Created Entries
	Updated Entries
	Dedups  int
	Deleted int

	NotModified int

	remoteEntries int
	refreshed     bool
	forceUpdate   bool
}

func NewFeedRefreshed() *FeedRefreshed { return new(FeedRefreshed) }

func NewFeedNotModified(v int) *FeedRefreshed {
	return &FeedRefreshed{NotModified: v}
}

func (self *FeedRefreshed) WithForceUpdate(value bool) *FeedRefreshed {
	self.forceUpdate = value
	return self
}

func (self *FeedRefreshed) Append(feedID int64, feedEntries []*Entry,
	storedEntries []Entry,
) *FeedRefreshed {
	storedBy := mapStoredEntries(feedID, storedEntries)
	if len(storedBy) == 0 {
		self.Created = feedEntries
		return self
	}

	for _, e := range feedEntries {
		storedEntry, stored := storedBy[e.Hash]
		switch {
		case !stored:
			self.Created = append(self.Created, e)
		case e.FeedID != storedEntry.FeedID:
			if !self.forceUpdate && !e.Date.After(storedEntry.Date) {
				e.KeepImportedStatus(EntryStatusRead)
				self.Dedups++
			}
			self.Created = append(self.Created, e)
		case self.forceUpdate || e.Date.After(storedEntry.Date):
			self.Updated = append(self.Updated, e)
		default:
			e.KeepImportedStatus(storedEntry.Status)
		}
	}
	return self
}

func mapStoredEntries(feedID int64, entries []Entry) map[string]*Entry {
	byHash := make(map[string]*Entry, len(entries))
	for i := range entries {
		e := &entries[i]
		if _, ok := byHash[e.Hash]; !ok || e.FeedID == feedID {
			byHash[e.Hash] = e
		}
	}
	return byHash
}

func (self *FeedRefreshed) CreatedLen() int { return len(self.Created) }
func (self *FeedRefreshed) UpdatedLen() int { return len(self.Updated) }

func (self *FeedRefreshed) WithRefreshed(n int) *FeedRefreshed {
	self.remoteEntries = n
	self.refreshed = true
	return self
}

func (self *FeedRefreshed) Refreshed() bool    { return self.refreshed }
func (self *FeedRefreshed) RemoteEntries() int { return self.remoteEntries }
