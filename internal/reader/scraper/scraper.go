// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package scraper // import "miniflux.app/v2/internal/reader/scraper"

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"miniflux.app/v2/internal/reader/encoding"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/readability"
	"miniflux.app/v2/internal/urllib"

	"github.com/PuerkitoBio/goquery"
)

func ScrapeWebsite(requestBuilder *fetcher.RequestBuilder, pageURL, rules string) (baseURL string, extractedContent string, err error) {
	responseHandler, err := requestBuilder.Request(pageURL)
	if err != nil {
		return "", "", fmt.Errorf("reader/scraper: scrape website: %w", err)
	}
	defer responseHandler.Close()

	if localizedError := responseHandler.LocalizedError(); localizedError != nil {
		slog.Warn("Unable to scrape website", slog.String("website_url", pageURL), slog.Any("error", localizedError))
		return "", "", localizedError
	}

	if !isAllowedContentType(responseHandler.ContentType()) {
		return "", "", fmt.Errorf("scraper: this resource is not a HTML document (%s)", responseHandler.ContentType())
	}

	// The entry URL could redirect somewhere else.
	sameSite := urllib.Domain(pageURL) == urllib.Domain(responseHandler.EffectiveURL())
	pageURL = responseHandler.EffectiveURL()

	if rules == "" {
		rules = getPredefinedScraperRules(pageURL)
	}

	htmlDocumentReader, err := encoding.NewCharsetReader(responseHandler.Body(),
		responseHandler.ContentType())
	if err != nil {
		return "", "", fmt.Errorf(
			"scraper: unable to read HTML document with charset reader: %w", err)
	}

	if sameSite && rules != "" {
		slog.Debug("Extracting content with custom rules",
			slog.String("url", pageURL), slog.String("rules", rules))
		baseURL, extractedContent, err = findContentUsingCustomRules(
			htmlDocumentReader, rules)
	} else {
		slog.Debug("Extracting content with readability",
			slog.String("url", pageURL))
		baseURL, extractedContent, err = readability.ExtractContent(
			htmlDocumentReader)
	}
	if err != nil {
		return "", "", fmt.Errorf("scraper: extract content: %w", err)
	}

	if baseURL == "" {
		baseURL = pageURL
	} else {
		slog.Debug("Using base URL from HTML document",
			slog.String("base_url", baseURL))
	}

	return baseURL, extractedContent, nil
}

func findContentUsingCustomRules(page io.Reader, rules string) (baseURL string, extractedContent string, err error) {
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
