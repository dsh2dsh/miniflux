// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package form // import "miniflux.app/v2/internal/ui/form"

import (
	"net/http"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/model"
)

// FeedForm represents a feed form in the UI
type FeedForm struct {
	FeedURL                     string
	SiteURL                     string
	Title                       string
	CommentsURLTemplate         string
	Description                 string
	ScraperRules                string
	RewriteRules                string
	UrlRewriteRules             string
	BlockAuthors                []string
	BlockFilterEntryRules       string
	KeepFilterEntryRules        string
	Crawler                     bool
	UserAgent                   string
	Cookie                      string
	CategoryID                  int64
	Username                    string
	Password                    string
	IgnoreHTTPCache             bool
	AllowSelfSignedCertificates bool
	FetchViaProxy               bool
	Disabled                    bool
	NoMediaPlayer               bool
	HideGlobally                bool
	CategoryHidden              bool // Category has "hide_globally"
	AppriseServiceURLs          string
	WebhookURL                  string
	DisableHTTP2                bool
	NtfyEnabled                 bool
	NtfyPriority                int
	NtfyTopic                   string
	PushoverEnabled             bool
	PushoverPriority            int
	ProxyURL                    string
}

func (self *FeedForm) BlockAuthorsFrom(s string) {
	if strings.TrimSpace(s) == "" {
		self.BlockAuthors = nil
		return
	}

	authors := make([]string, 0, strings.Count(s, "\n")+1)
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		authors = append(authors, line)
	}

	if len(authors) == 0 {
		self.BlockAuthors = nil
		return
	}
	self.BlockAuthors = authors
}

func (self *FeedForm) BlockAuthorsString() string {
	if len(self.BlockAuthors) == 0 {
		return ""
	}
	return strings.Join(self.BlockAuthors, "\n")
}

// Merge updates the fields of the given feed.
func (self *FeedForm) Merge(feed *model.Feed) *model.Feed {
	feed.Category.ID = self.CategoryID
	feed.Title = self.Title
	feed.SiteURL = self.SiteURL
	feed.FeedURL = self.FeedURL
	feed.Description = self.Description
	feed.ScraperRules = self.ScraperRules
	feed.RewriteRules = self.RewriteRules
	feed.UrlRewriteRules = self.UrlRewriteRules
	feed.WithBlockAuthors(self.BlockAuthors)
	feed.Extra.BlockFilterEntryRules = self.BlockFilterEntryRules
	feed.Extra.KeepFilterEntryRules = self.KeepFilterEntryRules
	feed.Crawler = self.Crawler
	feed.UserAgent = self.UserAgent
	feed.Cookie = self.Cookie
	feed.ParsingErrorCount = 0
	feed.ParsingErrorMsg = ""
	feed.Username = self.Username
	feed.Password = self.Password
	feed.IgnoreHTTPCache = self.IgnoreHTTPCache
	feed.AllowSelfSignedCertificates = self.AllowSelfSignedCertificates
	feed.FetchViaProxy = self.FetchViaProxy
	feed.Disabled = self.Disabled
	feed.NoMediaPlayer = self.NoMediaPlayer
	feed.HideGlobally = self.HideGlobally
	feed.AppriseServiceURLs = self.AppriseServiceURLs
	feed.WebhookURL = self.WebhookURL
	feed.DisableHTTP2 = self.DisableHTTP2
	feed.NtfyEnabled = self.NtfyEnabled
	feed.NtfyPriority = self.NtfyPriority
	feed.NtfyTopic = self.NtfyTopic
	feed.PushoverEnabled = self.PushoverEnabled
	feed.PushoverPriority = self.PushoverPriority
	feed.ProxyURL = self.ProxyURL
	feed.Extra.CommentsURLTemplate = self.CommentsURLTemplate
	return feed
}

// NewFeedForm parses the HTTP request and returns a FeedForm
func NewFeedForm(r *http.Request) *FeedForm {
	categoryID, err := strconv.Atoi(r.FormValue("category_id"))
	if err != nil {
		categoryID = 0
	}

	ntfyPriority, err := strconv.Atoi(r.FormValue("ntfy_priority"))
	if err != nil {
		ntfyPriority = 0
	}

	pushoverPriority, err := strconv.Atoi(r.FormValue("pushover_priority"))
	if err != nil {
		pushoverPriority = 0
	}

	ff := &FeedForm{
		FeedURL:                     r.FormValue("feed_url"),
		SiteURL:                     r.FormValue("site_url"),
		Title:                       r.FormValue("title"),
		CommentsURLTemplate:         r.FormValue("comments_url_template"),
		Description:                 r.FormValue("description"),
		ScraperRules:                r.FormValue("scraper_rules"),
		UserAgent:                   r.FormValue("user_agent"),
		Cookie:                      r.FormValue("cookie"),
		RewriteRules:                r.FormValue("rewrite_rules"),
		UrlRewriteRules:             r.FormValue("urlrewrite_rules"),
		BlockFilterEntryRules:       r.FormValue("block_filter_entry_rules"),
		KeepFilterEntryRules:        r.FormValue("keep_filter_entry_rules"),
		Crawler:                     r.FormValue("crawler") == "1",
		CategoryID:                  int64(categoryID),
		Username:                    r.FormValue("feed_username"),
		Password:                    r.FormValue("feed_password"),
		IgnoreHTTPCache:             r.FormValue("ignore_http_cache") == "1",
		AllowSelfSignedCertificates: r.FormValue("allow_self_signed_certificates") == "1",
		FetchViaProxy:               r.FormValue("fetch_via_proxy") == "1",
		Disabled:                    r.FormValue("disabled") == "1",
		NoMediaPlayer:               r.FormValue("no_media_player") == "1",
		HideGlobally:                r.FormValue("hide_globally") == "1",
		AppriseServiceURLs:          r.FormValue("apprise_service_urls"),
		WebhookURL:                  r.FormValue("webhook_url"),
		DisableHTTP2:                r.FormValue("disable_http2") == "1",
		NtfyEnabled:                 r.FormValue("ntfy_enabled") == "1",
		NtfyPriority:                ntfyPriority,
		NtfyTopic:                   r.FormValue("ntfy_topic"),
		PushoverEnabled:             r.FormValue("pushover_enabled") == "1",
		PushoverPriority:            pushoverPriority,
		ProxyURL:                    r.FormValue("proxy_url"),
	}
	ff.BlockAuthorsFrom(r.FormValue("blockAuthors"))
	return ff
}
