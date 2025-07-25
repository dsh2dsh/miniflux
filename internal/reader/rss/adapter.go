// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rss // import "miniflux.app/v2/internal/reader/rss"

import (
	"html"
	"log/slog"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/date"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/urllib"
)

type RSSAdapter struct {
	rss *RSS
}

func NewRSSAdapter(rss *RSS) *RSSAdapter {
	return &RSSAdapter{rss}
}

func (r *RSSAdapter) BuildFeed(baseURL string) *model.Feed {
	feed := &model.Feed{
		Title:       html.UnescapeString(strings.TrimSpace(r.rss.Channel.Title)),
		FeedURL:     strings.TrimSpace(baseURL),
		SiteURL:     strings.TrimSpace(r.rss.Channel.Link),
		Description: strings.TrimSpace(r.rss.Channel.Description),
	}

	// Ensure the Site URL is absolute.
	if absoluteSiteURL, err := urllib.AbsoluteURL(baseURL, feed.SiteURL); err == nil {
		feed.SiteURL = absoluteSiteURL
	}

	// Try to find the feed URL from the Atom links.
	for _, atomLink := range r.rss.Channel.Links {
		atomLinkHref := strings.TrimSpace(atomLink.Href)
		if atomLinkHref != "" && atomLink.Rel == "self" {
			if absoluteFeedURL, err := urllib.AbsoluteURL(feed.FeedURL, atomLinkHref); err == nil {
				feed.FeedURL = absoluteFeedURL
				break
			}
		}
	}

	// Fallback to the site URL if the title is empty.
	if feed.Title == "" {
		feed.Title = feed.SiteURL
	}

	// Get TTL if defined.
	if r.rss.Channel.TTL != "" {
		if ttl, err := strconv.Atoi(r.rss.Channel.TTL); err == nil {
			feed.TTL = ttl
		}
	}

	// Get the feed icon URL if defined.
	if r.rss.Channel.Image != nil {
		if absoluteIconURL, err := urllib.AbsoluteURL(feed.SiteURL, r.rss.Channel.Image.URL); err == nil {
			feed.IconURL = absoluteIconURL
		}
	}

	for _, item := range r.rss.Channel.Items {
		entry := model.NewEntry()
		entry.Date = findEntryDate(&item)
		entry.Content = findEntryContent(&item)
		entry.AppendEnclosures(findEntryEnclosures(&item, feed.SiteURL))

		// Populate the entry URL.
		entryURL := findEntryURL(&item)
		if entryURL == "" {
			// Fallback to the first enclosure URL if it exists.
			if len(entry.Enclosures()) > 0 && entry.Enclosures()[0].URL != "" {
				entry.URL = entry.Enclosures()[0].URL
			} else {
				// Fallback to the feed URL if no entry URL is found.
				entry.URL = feed.SiteURL
			}
		} else {
			if absoluteEntryURL, err := urllib.AbsoluteURL(feed.SiteURL, entryURL); err == nil {
				entry.URL = absoluteEntryURL
			} else {
				entry.URL = entryURL
			}
		}

		// Populate the entry title.
		entry.Title = findEntryTitle(&item)
		if entry.Title == "" {
			entry.Title = sanitizer.TruncateHTML(entry.Content, 100)
			if entry.Title == "" {
				entry.Title = entry.URL
			}
		}

		entry.Author = findEntryAuthor(&item)
		if entry.Author == "" {
			entry.Author = findFeedAuthor(&r.rss.Channel)
		}

		// Generate the entry hash.
		switch {
		case item.GUID.Data != "":
			entry.Hash = crypto.HashFromStringCompat(item.GUID.Data, entry.Date)
		case entryURL != "":
			entry.Hash = crypto.HashFromStringCompat(entryURL, entry.Date)
		default:
			entry.Hash = crypto.HashFromStringCompat(entry.Title+entry.Content,
				entry.Date)
		}

		// Find CommentsURL if defined.
		if absoluteCommentsURL := strings.TrimSpace(item.CommentsURL); absoluteCommentsURL != "" && urllib.IsAbsoluteURL(absoluteCommentsURL) {
			entry.CommentsURL = absoluteCommentsURL
		}

		// Set podcast listening time.
		if item.ItunesDuration != "" {
			if duration, err := getDurationInMinutes(item.ItunesDuration); err == nil {
				entry.ReadingTime = duration
			}
		}

		// Populate entry categories.
		entry.Tags = findEntryTags(&item)
		if len(entry.Tags) == 0 {
			entry.Tags = findFeedTags(&r.rss.Channel)
		}
		// Sort and deduplicate tags.
		slices.Sort(entry.Tags)
		entry.Tags = slices.Compact(entry.Tags)

		feed.Entries = append(feed.Entries, entry)
	}

	return feed
}

func findFeedAuthor(rssChannel *RSSChannel) string {
	var author string
	switch {
	case rssChannel.ItunesAuthor != "":
		author = rssChannel.ItunesAuthor
	case rssChannel.GooglePlayAuthor != "":
		author = rssChannel.GooglePlayAuthor
	case rssChannel.ItunesOwner.String() != "":
		author = rssChannel.ItunesOwner.String()
	case rssChannel.ManagingEditor != "":
		author = rssChannel.ManagingEditor
	case rssChannel.Webmaster != "":
		author = rssChannel.Webmaster
	default:
		return ""
	}

	return strings.TrimSpace(sanitizer.StripTags(author))
}

func findFeedTags(rssChannel *RSSChannel) []string {
	tags := make([]string, 0)

	for _, tag := range rssChannel.Categories {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	for _, tag := range rssChannel.GetItunesCategories() {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	if tag := strings.TrimSpace(rssChannel.GooglePlayCategory.Text); tag != "" {
		tags = append(tags, tag)
	}

	return tags
}

func findEntryTitle(rssItem *RSSItem) string {
	title := rssItem.Title.Content

	if rssItem.DublinCoreTitle != "" {
		title = rssItem.DublinCoreTitle
	}

	return html.UnescapeString(html.UnescapeString(strings.TrimSpace(title)))
}

func findEntryURL(rssItem *RSSItem) string {
	for _, link := range []string{rssItem.FeedBurnerLink, rssItem.Link} {
		if link != "" {
			return strings.TrimSpace(link)
		}
	}

	for _, atomLink := range rssItem.Links {
		if atomLink.Href != "" && (strings.EqualFold(atomLink.Rel, "alternate") || atomLink.Rel == "") {
			return strings.TrimSpace(atomLink.Href)
		}
	}

	// Specs: https://cyber.harvard.edu/rss/rss.html#ltguidgtSubelementOfLtitemgt
	// isPermaLink is optional, its default value is true.
	// If its value is false, the guid may not be assumed to be a url, or a url to anything in particular.
	if rssItem.GUID.IsPermaLink == "true" || rssItem.GUID.IsPermaLink == "" {
		return strings.TrimSpace(rssItem.GUID.Data)
	}

	return ""
}

func findEntryContent(rssItem *RSSItem) string {
	for _, value := range []string{
		rssItem.DublinCoreContent,
		rssItem.Description,
		rssItem.GooglePlayDescription,
		rssItem.ItunesSummary,
		rssItem.ItunesSubtitle,
	} {
		if value != "" {
			return value
		}
	}
	return ""
}

func findEntryDate(rssItem *RSSItem) time.Time {
	value := rssItem.PubDate
	if rssItem.DublinCoreDate != "" {
		value = rssItem.DublinCoreDate
	}

	if value != "" {
		result, err := date.Parse(value)
		if err != nil {
			slog.Debug("Unable to parse date from RSS feed",
				slog.String("date", value),
				slog.String("guid", rssItem.GUID.Data),
				slog.Any("error", err),
			)
			return time.Now()
		}

		return result
	}

	return time.Now()
}

func findEntryAuthor(rssItem *RSSItem) string {
	var author string

	switch {
	case rssItem.GooglePlayAuthor != "":
		author = rssItem.GooglePlayAuthor
	case rssItem.ItunesAuthor != "":
		author = rssItem.ItunesAuthor
	case rssItem.DublinCoreCreator != "":
		author = rssItem.DublinCoreCreator
	case rssItem.PersonName() != "":
		author = rssItem.PersonName()
	case strings.Contains(rssItem.Author.Inner, "<![CDATA["):
		author = rssItem.Author.Data
	case rssItem.Author.Inner != "":
		author = rssItem.Author.Inner
	default:
		return ""
	}

	return strings.TrimSpace(sanitizer.StripTags(author))
}

func findEntryTags(rssItem *RSSItem) []string {
	tags := make([]string, 0)

	for _, tag := range rssItem.Categories {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	for _, tag := range rssItem.MediaCategories.Labels() {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags
}

func findEntryEnclosures(rssItem *RSSItem, siteURL string) model.EnclosureList {
	enclosures := make(model.EnclosureList, 0)
	duplicates := make(map[string]bool)

	mediaThumbnails := rssItem.AllMediaThumbnails()
	enclosures = slices.Grow(enclosures, len(mediaThumbnails))
	for _, mediaThumbnail := range mediaThumbnails {
		mediaURL := strings.TrimSpace(mediaThumbnail.URL)
		if mediaURL == "" {
			continue
		}
		if _, found := duplicates[mediaURL]; !found {
			if mediaAbsoluteURL, err := urllib.AbsoluteURL(siteURL, mediaURL); err != nil {
				slog.Debug("Unable to build absolute URL for media thumbnail",
					slog.String("url", mediaThumbnail.URL),
					slog.String("site_url", siteURL),
					slog.Any("error", err),
				)
			} else {
				duplicates[mediaAbsoluteURL] = true
				enclosures = append(enclosures, model.Enclosure{
					URL:      mediaAbsoluteURL,
					MimeType: mediaThumbnail.MimeType(),
					Size:     mediaThumbnail.Size(),
				})
			}
		}
	}

	enclosures = slices.Grow(enclosures, len(rssItem.Enclosures))
	for _, enclosure := range rssItem.Enclosures {
		enclosureURL := enclosure.URL

		if rssItem.FeedBurnerEnclosureLink != "" {
			filename := path.Base(rssItem.FeedBurnerEnclosureLink)
			if strings.HasSuffix(enclosureURL, filename) {
				enclosureURL = rssItem.FeedBurnerEnclosureLink
			}
		}

		enclosureURL = strings.TrimSpace(enclosureURL)
		if enclosureURL == "" {
			continue
		}

		if absoluteEnclosureURL, err := urllib.AbsoluteURL(siteURL, enclosureURL); err == nil {
			enclosureURL = absoluteEnclosureURL
		}

		if _, found := duplicates[enclosureURL]; !found {
			duplicates[enclosureURL] = true

			enclosures = append(enclosures, model.Enclosure{
				URL:      enclosureURL,
				MimeType: enclosure.Type,
				Size:     enclosure.Size(),
			})
		}
	}

	mediaContents := rssItem.AllMediaContents()
	enclosures = slices.Grow(enclosures, len(mediaContents))
	for _, mediaContent := range mediaContents {
		mediaURL := strings.TrimSpace(mediaContent.URL)
		if mediaURL == "" {
			continue
		}
		if _, found := duplicates[mediaURL]; !found {
			mediaURL := strings.TrimSpace(mediaContent.URL)
			if mediaAbsoluteURL, err := urllib.AbsoluteURL(siteURL, mediaURL); err != nil {
				slog.Debug("Unable to build absolute URL for media content",
					slog.String("url", mediaContent.URL),
					slog.String("site_url", siteURL),
					slog.Any("error", err),
				)
			} else {
				duplicates[mediaAbsoluteURL] = true
				enclosures = append(enclosures, model.Enclosure{
					URL:      mediaAbsoluteURL,
					MimeType: mediaContent.MimeType(),
					Size:     mediaContent.Size(),
				})
			}
		}
	}

	mediaPeerLinks := rssItem.AllMediaPeerLinks()
	enclosures = slices.Grow(enclosures, len(mediaPeerLinks))
	for _, mediaPeerLink := range mediaPeerLinks {
		mediaURL := strings.TrimSpace(mediaPeerLink.URL)
		if mediaURL == "" {
			continue
		}
		if _, found := duplicates[mediaURL]; !found {
			mediaURL := strings.TrimSpace(mediaPeerLink.URL)
			if mediaAbsoluteURL, err := urllib.AbsoluteURL(siteURL, mediaURL); err != nil {
				slog.Debug("Unable to build absolute URL for media peer link",
					slog.String("url", mediaPeerLink.URL),
					slog.String("site_url", siteURL),
					slog.Any("error", err),
				)
			} else {
				duplicates[mediaAbsoluteURL] = true
				enclosures = append(enclosures, model.Enclosure{
					URL:      mediaAbsoluteURL,
					MimeType: mediaPeerLink.MimeType(),
					Size:     mediaPeerLink.Size(),
				})
			}
		}
	}

	return enclosures
}
