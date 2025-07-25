// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"fmt"
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

func NewFeed() *Feed {
	return &Feed{Category: &Category{}, Icon: &FeedIcon{}}
}

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

	removedByAge     int
	removedByFilters int
	removedByHash    int
}

type FeedExtra struct {
	CommentsURLTemplate string `json:"comments_url_template,omitempty"`

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

func (f *Feed) String() string {
	return fmt.Sprintf("ID=%d, UserID=%d, FeedURL=%s, SiteURL=%s, Title=%s, Category={%s}",
		f.ID,
		f.UserID,
		f.FeedURL,
		f.SiteURL,
		f.Title,
		f.Category,
	)
}

// WithCategoryID initializes the category attribute of the feed.
func (f *Feed) WithCategoryID(categoryID int64) {
	f.Category = &Category{ID: categoryID}
}

// WithTranslatedErrorMessage adds a new error message and increment the error counter.
func (f *Feed) WithTranslatedErrorMessage(message string) {
	f.ParsingErrorCount++
	f.ParsingErrorMsg = message
}

// ResetErrorCounter removes all previous errors.
func (f *Feed) ResetErrorCounter() {
	f.ParsingErrorCount, f.ParsingErrorMsg = 0, ""
}

// CheckedNow set attribute values when the feed is refreshed.
func (f *Feed) CheckedNow() {
	f.CheckedAt = time.Now()
	if f.SiteURL == "" {
		f.SiteURL = f.FeedURL
	}
}

// ScheduleNextCheck set "next_check_at" of a feed.
func (f *Feed) ScheduleNextCheck(refreshDelayInMinutes int) int {
	// Default to the global config Polling Frequency.
	//
	// Use the RSS TTL field, Retry-After, Cache-Control or Expires HTTP headers
	// if defined.
	intervalMinutes := max(config.Opts.SchedulerRoundRobinMinInterval(),
		refreshDelayInMinutes)

	// Limit the max interval value for misconfigured feeds.
	intervalMinutes = min(config.Opts.SchedulerRoundRobinMaxInterval(),
		intervalMinutes)

	f.NextCheckAt = time.Now().Add(time.Minute * time.Duration(intervalMinutes))
	return intervalMinutes
}

func (f *Feed) Size() uint64 { return f.Runtime.Size }
func (f *Feed) HashString() string {
	return strconv.FormatUint(f.Runtime.Hash, 16)
}

func (f *Feed) ContentChanged(body []byte) bool {
	oldSize, oldHash := f.Runtime.Size, f.Runtime.Hash
	f.Runtime.Size, f.Runtime.Hash = uint64(len(body)), xxhash.Sum64(body)
	return f.Runtime.Size != oldSize || f.Runtime.Hash != oldHash
}

func (f *Feed) CommentsURLTemplate() (*template.Template, error) {
	s := strings.TrimSpace(f.Extra.CommentsURLTemplate)
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

func (f *Feed) BlockFilterEntryRules() string {
	return f.Extra.BlockFilterEntryRules
}

func (f *Feed) KeepFilterEntryRules() string {
	return f.Extra.KeepFilterEntryRules
}

func (f *Feed) IncRemovedByAge()  { f.removedByAge++ }
func (f *Feed) RemovedByAge() int { return f.removedByAge }

func (f *Feed) IncRemovedByFilters()  { f.removedByFilters++ }
func (f *Feed) RemovedByFilters() int { return f.removedByFilters }

func (f *Feed) IncRemovedByHash()  { f.removedByHash++ }
func (f *Feed) RemovedByHash() int { return f.removedByHash }

// FeedCreationRequest represents the request to create a feed.
type FeedCreationRequest struct {
	FeedURL                     string `json:"feed_url"`
	CategoryID                  int64  `json:"category_id"`
	UserAgent                   string `json:"user_agent"`
	Cookie                      string `json:"cookie"`
	Username                    string `json:"username"`
	Password                    string `json:"password"`
	Crawler                     bool   `json:"crawler"`
	Disabled                    bool   `json:"disabled"`
	NoMediaPlayer               bool   `json:"no_media_player"`
	IgnoreHTTPCache             bool   `json:"ignore_http_cache"`
	AllowSelfSignedCertificates bool   `json:"allow_self_signed_certificates"`
	FetchViaProxy               bool   `json:"fetch_via_proxy"`
	HideGlobally                bool   `json:"hide_globally"`
	DisableHTTP2                bool   `json:"disable_http2"`
	ScraperRules                string `json:"scraper_rules"`
	RewriteRules                string `json:"rewrite_rules"`
	BlockFilterEntryRules       string `json:"block_filter_entry_rules"`
	KeepFilterEntryRules        string `json:"keep_filter_entry_rules"`
	UrlRewriteRules             string `json:"urlrewrite_rules"`
	ProxyURL                    string `json:"proxy_url"`
}

type FeedCreationRequestFromSubscriptionDiscovery struct {
	FeedCreationRequest

	Content      []byte
	ETag         string
	LastModified string
}

// FeedModificationRequest represents the request to update a feed.
type FeedModificationRequest struct {
	FeedURL                     *string `json:"feed_url"`
	SiteURL                     *string `json:"site_url"`
	Title                       *string `json:"title"`
	Description                 *string `json:"description"`
	ScraperRules                *string `json:"scraper_rules"`
	RewriteRules                *string `json:"rewrite_rules"`
	UrlRewriteRules             *string `json:"urlrewrite_rules"`
	BlockFilterEntryRules       *string `json:"block_filter_entry_rules"`
	KeepFilterEntryRules        *string `json:"keep_filter_entry_rules"`
	Crawler                     *bool   `json:"crawler"`
	UserAgent                   *string `json:"user_agent"`
	Cookie                      *string `json:"cookie"`
	Username                    *string `json:"username"`
	Password                    *string `json:"password"`
	CategoryID                  *int64  `json:"category_id"`
	Disabled                    *bool   `json:"disabled"`
	NoMediaPlayer               *bool   `json:"no_media_player"`
	IgnoreHTTPCache             *bool   `json:"ignore_http_cache"`
	AllowSelfSignedCertificates *bool   `json:"allow_self_signed_certificates"`
	FetchViaProxy               *bool   `json:"fetch_via_proxy"`
	HideGlobally                *bool   `json:"hide_globally"`
	DisableHTTP2                *bool   `json:"disable_http2"`
	ProxyURL                    *string `json:"proxy_url"`
	CommentsURLTemplate         *string `json:"comments_url_template,omitempty"`
}

// Patch updates a feed with modified values.
func (f *FeedModificationRequest) Patch(feed *Feed) {
	if f.FeedURL != nil && *f.FeedURL != "" {
		feed.FeedURL = *f.FeedURL
	}

	if f.SiteURL != nil && *f.SiteURL != "" {
		feed.SiteURL = *f.SiteURL
	}

	if f.Title != nil && *f.Title != "" {
		feed.Title = *f.Title
	}

	if f.Description != nil && *f.Description != "" {
		feed.Description = *f.Description
	}

	if f.ScraperRules != nil {
		feed.ScraperRules = *f.ScraperRules
	}

	if f.RewriteRules != nil {
		feed.RewriteRules = *f.RewriteRules
	}

	if f.UrlRewriteRules != nil {
		feed.UrlRewriteRules = *f.UrlRewriteRules
	}

	if f.BlockFilterEntryRules != nil {
		feed.Extra.BlockFilterEntryRules = *f.BlockFilterEntryRules
	}

	if f.KeepFilterEntryRules != nil {
		feed.Extra.KeepFilterEntryRules = *f.KeepFilterEntryRules
	}

	if f.Crawler != nil {
		feed.Crawler = *f.Crawler
	}

	if f.UserAgent != nil {
		feed.UserAgent = *f.UserAgent
	}

	if f.Cookie != nil {
		feed.Cookie = *f.Cookie
	}

	if f.Username != nil {
		feed.Username = *f.Username
	}

	if f.Password != nil {
		feed.Password = *f.Password
	}

	if f.CategoryID != nil && *f.CategoryID > 0 {
		feed.Category.ID = *f.CategoryID
	}

	if f.Disabled != nil {
		feed.Disabled = *f.Disabled
	}

	if f.NoMediaPlayer != nil {
		feed.NoMediaPlayer = *f.NoMediaPlayer
	}

	if f.IgnoreHTTPCache != nil {
		feed.IgnoreHTTPCache = *f.IgnoreHTTPCache
	}

	if f.AllowSelfSignedCertificates != nil {
		feed.AllowSelfSignedCertificates = *f.AllowSelfSignedCertificates
	}

	if f.FetchViaProxy != nil {
		feed.FetchViaProxy = *f.FetchViaProxy
	}

	if f.HideGlobally != nil {
		feed.HideGlobally = *f.HideGlobally
	}

	if f.DisableHTTP2 != nil {
		feed.DisableHTTP2 = *f.DisableHTTP2
	}

	if f.ProxyURL != nil {
		feed.ProxyURL = *f.ProxyURL
	}

	if f.CommentsURLTemplate != nil {
		feed.Extra.CommentsURLTemplate = *f.CommentsURLTemplate
	}
}

// Feeds is a list of feed
type Feeds []*Feed

type FeedRefreshed struct {
	CreatedEntries Entries
	UpdatedEntires Entries

	Dedups  uint64
	Deleted uint64

	Refreshed   bool
	NotModified int
}

func NewFeedRefreshed() *FeedRefreshed { return new(FeedRefreshed) }

func (self *FeedRefreshed) Append(entries []*Entry, published map[string]*Entry,
) *FeedRefreshed {
	if len(published) == 0 {
		self.CreatedEntries = entries
		return self
	}

	for _, e := range entries {
		stored, ok := published[e.Hash]
		switch {
		case !ok:
			e.Status = EntryStatusUnread
			self.CreatedEntries = append(self.CreatedEntries, e)
		case e.FeedID != stored.FeedID:
			if e.Date.After(stored.Date) {
				e.Status = EntryStatusUnread
			} else {
				e.Status = EntryStatusRead
				self.Dedups++
			}
			self.CreatedEntries = append(self.CreatedEntries, e)
		case e.Date.After(stored.Date):
			e.Status = EntryStatusUnread
			self.UpdatedEntires = append(self.UpdatedEntires, e)
		default:
			e.Status = stored.Status
		}
	}
	return self
}

func (self *FeedRefreshed) Created() int { return len(self.CreatedEntries) }
func (self *FeedRefreshed) Updated() int { return len(self.UpdatedEntires) }
