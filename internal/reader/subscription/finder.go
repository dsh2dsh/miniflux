// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package subscription // import "miniflux.app/v2/internal/reader/subscription"

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"miniflux.app/v2/internal/integration/rssbridge"
	"miniflux.app/v2/internal/locale"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/parser"
	"miniflux.app/v2/internal/urllib"
)

type SubscriptionFinder struct {
	requestBuilder   *fetcher.RequestBuilder
	feedDownloaded   bool
	feedResponseInfo *model.FeedCreationRequestFromSubscriptionDiscovery
}

func NewSubscriptionFinder(requestBuilder *fetcher.RequestBuilder) *SubscriptionFinder {
	return &SubscriptionFinder{
		requestBuilder: requestBuilder,
	}
}

func (f *SubscriptionFinder) IsFeedAlreadyDownloaded() bool {
	return f.feedDownloaded
}

func (f *SubscriptionFinder) FeedResponseInfo() *model.FeedCreationRequestFromSubscriptionDiscovery {
	return f.feedResponseInfo
}

func (f *SubscriptionFinder) FindSubscriptions(ctx context.Context,
	websiteURL, rssBridgeURL, rssBridgeToken string,
) (Subscriptions, *locale.LocalizedErrorWrapper) {
	resp, err := f.requestBuilder.RequestWithContext(ctx, websiteURL)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.http_body_read", err)
	}
	defer resp.Close()

	log := logging.FromContext(ctx).With(slog.String("website_url", websiteURL))

	if lerr := resp.LocalizedError(); lerr != nil {
		log.Warn("Unable to find subscriptions", slog.Any("error", lerr))
		return nil, lerr
	}

	body, lerr := resp.ReadBody()
	if lerr != nil {
		log.Warn("Unable to find subscriptions", slog.Any("error", lerr))
		return nil, lerr
	}
	resp.Close()

	f.feedResponseInfo = &model.FeedCreationRequestFromSubscriptionDiscovery{
		Content:      body,
		ETag:         resp.ETag(),
		LastModified: resp.LastModified(),
	}

	// Step 1) Check if the website URL is already a feed.
	if feed, err := parser.ParseBytes(resp.EffectiveURL(), body); err == nil {
		f.feedDownloaded = true
		s := NewSubscription(feed.Title, resp.EffectiveURL())
		if s.Title == "" {
			s.Title = resp.EffectiveURL()
		}
		return Subscriptions{s}, nil
	}

	// Step 2) Find the canonical URL of the website.
	log.Debug("Try to find the canonical URL of the website")
	websiteURL = f.findCanonicalURL(websiteURL, resp.ContentType(), body)

	// Step 3) Check if the website URL is a YouTube channel.
	log.Debug("Try to detect feeds for a YouTube page")
	subscriptions, lerr := f.findSubscriptionsFromYouTube(log, websiteURL)
	if lerr != nil {
		return nil, lerr
	}

	subscriptions = subscriptions.Parseable(f.requestBuilder)
	if len(subscriptions) > 0 {
		log.Debug("Subscriptions found from YouTube page",
			slog.Any("subscriptions", subscriptions))
		return subscriptions, nil
	}

	// Step 4) Parse web page to find feeds from HTML meta tags.
	log.Debug("Try to detect feeds from HTML meta tags",
		slog.String("content_type", resp.ContentType()))
	subscriptions, lerr = f.findSubscriptionsFromWebPage(websiteURL,
		resp.ContentType(), body)
	if lerr != nil {
		return nil, lerr
	}

	subscriptions = subscriptions.Parseable(f.requestBuilder)
	if len(subscriptions) > 0 {
		log.Debug("Subscriptions found from web page",
			slog.Any("subscriptions", subscriptions))
		return subscriptions, nil
	}

	// Step 5) Check if the website URL can use RSS-Bridge.
	if rssBridgeURL != "" {
		log.Debug("Try to detect feeds with RSS-Bridge")
		subscriptions, lerr := f.findSubscriptionsFromRSSBridge(log, websiteURL,
			rssBridgeURL, rssBridgeToken)
		if lerr != nil {
			return nil, lerr
		}

		subscriptions = subscriptions.Parseable(f.requestBuilder)
		if len(subscriptions) > 0 {
			log.Debug("Subscriptions found from RSS-Bridge",
				slog.Any("subscriptions", subscriptions))
			return subscriptions, nil
		}
	}

	// Step 6) Check if the website has a known feed URL.
	log.Debug("Try to detect feeds from well-known URLs")
	subscriptions = f.findSubscriptionsFromWellKnownURLs(websiteURL)

	// Some websites redirects unknown URLs to the home page.
	// As result, the list of known URLs is returned to the subscription list.
	// We don't want the user to choose between invalid feed URLs.
	f.requestBuilder.WithoutRedirects()

	subscriptions = subscriptions.Parseable(f.requestBuilder)
	if len(subscriptions) > 0 {
		log.Debug("Subscriptions found with well-known URLs",
			slog.Any("subscriptions", subscriptions))
		return subscriptions, nil
	}
	return nil, nil
}

func (f *SubscriptionFinder) findSubscriptionsFromWebPage(websiteURL,
	contentType string, body []byte,
) (Subscriptions, *locale.LocalizedErrorWrapper) {
	queries := [...]string{
		"link[type='application/rss+xml']",
		"link[type='application/atom+xml']",
		"link[type='application/json'], link[type='application/feed+json']",
	}

	htmlDocumentReader, err := encoding.NewCharsetReader(bytes.NewReader(body),
		contentType)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_html_document", err)
	}

	doc, err := goquery.NewDocumentFromReader(htmlDocumentReader)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_parse_html_document", err)
	}

	hrefValue, exists := doc.FindMatcher(goquery.Single("head base")).Attr("href")
	if exists {
		hrefValue = strings.TrimSpace(hrefValue)
		if urllib.IsAbsoluteURL(hrefValue) {
			websiteURL = hrefValue
		}
	}

	var subscriptions Subscriptions
	subscriptionURLs := make(map[string]bool)
	for _, query := range queries {
		doc.Find(query).Each(func(i int, s *goquery.Selection) {
			subscription := NewSubscription("", "")

			if feedURL, exists := s.Attr("href"); exists && feedURL != "" {
				subscription.URL, err = urllib.ResolveToAbsoluteURL(websiteURL, feedURL)
				if err != nil {
					return
				}
			} else {
				return // without an url, there can be no subscription.
			}

			if title, exists := s.Attr("title"); exists {
				subscription.Title = title
			}

			if subscription.Title == "" {
				subscription.Title = subscription.URL
			}

			if !subscriptionURLs[subscription.URL] {
				subscriptionURLs[subscription.URL] = true
				subscriptions = append(subscriptions, subscription)
			}
		})
	}
	return subscriptions, nil
}

func (f *SubscriptionFinder) findSubscriptionsFromWellKnownURLs(
	websiteURL string,
) Subscriptions {
	knownURLs := [...]string{
		"atom.xml",
		"feed.atom",
		"feed.xml",
		"feed/",
		"index.rss",
		"index.xml",
		"rss.xml",
		"rss/",
		"rss/feed.xml",
	}

	websiteURLRoot := urllib.RootURL(websiteURL)
	baseURLs := []string{
		// Look for knownURLs in the root.
		websiteURLRoot,
	}

	// Look for knownURLs in current subdirectory, such as 'example.com/blog/'.
	websiteURL, _ = urllib.ResolveToAbsoluteURL(websiteURL, "./")
	if websiteURL != websiteURLRoot {
		baseURLs = append(baseURLs, websiteURL)
	}

	subscriptions := make(Subscriptions, 0, len(baseURLs)*len(knownURLs))
	for _, baseURL := range baseURLs {
		for _, knownURL := range knownURLs {
			fullURL, err := urllib.ResolveToAbsoluteURL(baseURL, knownURL)
			if err != nil {
				continue
			}
			subscriptions = append(subscriptions, NewSubscription(fullURL, fullURL))
		}
	}
	return subscriptions
}

func (f *SubscriptionFinder) findSubscriptionsFromRSSBridge(log *slog.Logger,
	websiteURL, rssBridgeURL, rssBridgeToken string,
) (Subscriptions, *locale.LocalizedErrorWrapper) {
	log = log.With(
		slog.String("rssbridge_url", rssBridgeURL),
		slog.String("rssbridge_token", rssBridgeToken))
	log.Debug("Trying to detect feeds using RSS-Bridge")

	bridges, err := rssbridge.DetectBridges(rssBridgeURL, rssBridgeToken,
		websiteURL)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err,
			"error.unable_to_detect_rssbridge", err)
	}

	log.Debug("RSS-Bridge results", slog.Int("nb_bridges", len(bridges)))

	if len(bridges) == 0 {
		return nil, nil
	}

	subscriptions := make(Subscriptions, len(bridges))
	for i, bridge := range bridges {
		subscriptions[i] = NewSubscription(bridge.BridgeMeta.Name, bridge.URL)
	}
	return subscriptions, nil
}

func (f *SubscriptionFinder) findSubscriptionsFromYouTube(log *slog.Logger,
	websiteURL string,
) (Subscriptions, *locale.LocalizedErrorWrapper) {
	playlistPrefixes := [...]struct {
		prefix string
		title  string
	}{
		{"UULF", "Videos"},
		{"UUSH", "Short videos"},
		{"UULV", "Live streams"},
	}

	decodedURL, err := url.Parse(websiteURL)
	if err != nil {
		return nil, locale.NewLocalizedErrorWrapper(err, "error.invalid_site_url", err)
	}

	if !strings.HasSuffix(decodedURL.Hostname(), "youtube.com") {
		log.Debug("YouTube feed discovery skipped: not a YouTube domain")
		return nil, nil
	}

	const videosXML = "https://www.youtube.com/feeds/videos.xml"
	if _, baseID, found := strings.Cut(decodedURL.Path, "channel/UC"); found {
		subscriptions := make(Subscriptions, 0, len(playlistPrefixes)+1)
		channelFeedURL := videosXML + "?channel_id=UC" + baseID
		subscriptions = append(subscriptions,
			NewSubscription("Channel", channelFeedURL))

		for _, playlist := range playlistPrefixes {
			playlistFeedURL := videosXML + "?playlist_id=" + playlist.prefix + baseID
			subscriptions = append(subscriptions,
				NewSubscription(playlist.title, playlistFeedURL))
		}
		return subscriptions, nil
	}

	playlist := strings.HasPrefix(decodedURL.EscapedPath(), "/watch") ||
		strings.HasPrefix(decodedURL.EscapedPath(), "/playlist")
	if !playlist {
		return nil, nil
	}

	if playlistID := decodedURL.Query().Get("list"); playlistID != "" {
		feedURL := videosXML + "?playlist_id=" + playlistID
		subscriptions := Subscriptions{NewSubscription(decodedURL.String(), feedURL)}
		return subscriptions, nil
	}
	return nil, nil
}

// findCanonicalURL extracts the canonical URL from the HTML <link rel="canonical"> tag.
// Returns the canonical URL if found, otherwise returns the effective URL.
func (f *SubscriptionFinder) findCanonicalURL(effectiveURL, contentType string,
	body []byte,
) string {
	htmlDocumentReader, err := encoding.NewCharsetReader(bytes.NewReader(body),
		contentType)
	if err != nil {
		return effectiveURL
	}

	doc, err := goquery.NewDocumentFromReader(htmlDocumentReader)
	if err != nil {
		return effectiveURL
	}

	baseURL := effectiveURL
	hrefValue, exists := doc.FindMatcher(goquery.Single("head base")).Attr("href")
	if exists {
		hrefValue = strings.TrimSpace(hrefValue)
		if urllib.IsAbsoluteURL(hrefValue) {
			baseURL = hrefValue
		}
	}

	canonicalHref, exists := doc.Find("link[rel='canonical' i]").First().
		Attr("href")
	if !exists || strings.TrimSpace(canonicalHref) == "" {
		return effectiveURL
	}

	canonicalURL, err := urllib.ResolveToAbsoluteURL(baseURL,
		strings.TrimSpace(canonicalHref))
	if err != nil {
		return effectiveURL
	}
	return canonicalURL
}
