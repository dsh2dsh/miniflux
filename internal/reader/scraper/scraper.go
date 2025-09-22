// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scraper // import "miniflux.app/v2/internal/reader/scraper"

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"

	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

func ScrapeWebsite(ctx context.Context, requestBuilder *fetcher.RequestBuilder,
	pageURL, rules string,
) (string, string, error) {
	resp, err := requestBuilder.Request(pageURL)
	if err != nil {
		return "", "", fmt.Errorf("reader/scraper: scrape website: %w", err)
	}
	defer resp.Close()

	log := logging.FromContext(ctx)
	if lerr := resp.LocalizedError(); lerr != nil {
		log.Warn("Unable to scrape website",
			slog.String("url", pageURL),
			slog.Any("error", lerr))
		return "", "", lerr
	}

	if !isAllowedContentType(resp.ContentType()) {
		return "", "", fmt.Errorf(
			"reader/scraper: this resource is not a HTML document (%s)",
			resp.ContentType())
	}

	r, err := encoding.NewCharsetReader(resp.Body(),
		resp.ContentType())
	if err != nil {
		return "", "", fmt.Errorf(
			"reader/scraper: unable to read HTML document with charset reader: %w",
			err)
	}

	// The entry URL could redirect somewhere else.
	sameSite := urllib.Domain(pageURL) == urllib.Domain(resp.EffectiveURL())
	if rules == "" {
		rules = getPredefinedScraperRules(resp.EffectiveURL())
	}

	if sameSite && rules != "" {
		return extractCustom(ctx, r, resp.URL(), rules)
	}
	return extractReadability(ctx, r, resp.URL())
}

func extractCustom(ctx context.Context, r io.Reader, u *url.URL, rules string,
) (string, string, error) {
	pageURL := u.String()
	log := logging.FromContext(ctx).With(slog.String("url", pageURL))
	log.Debug("Extracting content with custom rules", slog.String("rules", rules))

	contentURL, content, err := findContentUsingCustomRules(r, rules)
	if err != nil {
		return "", "", fmt.Errorf(
			"reader/scraper: extracting custom content: %w", err)
	} else if contentURL == "" {
		return pageURL, content, nil
	}

	log.Debug("Using base URL from HTML document",
		slog.String("base_url", contentURL))
	return contentURL, content, nil
}

func extractReadability(ctx context.Context, r io.Reader, u *url.URL) (string, string, error) {
	pageURL := u.String()
	log := logging.FromContext(ctx).With(slog.String("url", pageURL))
	log.Debug("Extracting content with readability", slog.String("url", pageURL))

	article, err := readability.FromReader(r, u)
	if err != nil {
		return "", "", fmt.Errorf(
			"reader/scraper: extracting readable content: %w", err)
	}
	return pageURL, article.Content, nil
}

func findContentUsingCustomRules(page io.Reader, rules string) (baseURL, extractedContent string, err error) {
	document, err := goquery.NewDocumentFromReader(page)
	if err != nil {
		return "", "", fmt.Errorf("reader/scraper: %w", err)
	}

	if hrefValue, exists := document.FindMatcher(goquery.Single("head base")).Attr("href"); exists {
		hrefValue = strings.TrimSpace(hrefValue)
		if urllib.IsAbsoluteURL(hrefValue) {
			baseURL = hrefValue
		}
	}

	document.Find(rules).Each(func(i int, s *goquery.Selection) {
		if content, err := goquery.OuterHtml(s); err == nil {
			extractedContent += content
		}
	})

	return baseURL, extractedContent, nil
}

func getPredefinedScraperRules(websiteURL string) string {
	urlDomain := urllib.DomainWithoutWWW(websiteURL)

	if rules, ok := predefinedRules[urlDomain]; ok {
		return rules
	}
	return ""
}

func isAllowedContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "text/html") ||
		strings.HasPrefix(contentType, "application/xhtml+xml")
}
