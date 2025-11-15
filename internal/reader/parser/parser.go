// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package parser // import "miniflux.app/v2/internal/reader/parser"

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/dsh2dsh/gofeed/v2"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/atom"
	"miniflux.app/v2/internal/reader/json"
	"miniflux.app/v2/internal/reader/rdf"
	"miniflux.app/v2/internal/reader/rss"
)

var ErrFeedFormatNotDetected = errors.New("parser: unable to detect feed format")

// ParseFeed analyzes the input data and returns a normalized feed object.
func ParseFeed(baseURL string, r io.ReadSeeker) (*model.Feed, error) {
	format, version := DetectFeedFormat(r)
	switch format {
	case FormatAtom:
		return atom.Parse(baseURL, r, version)
	case FormatRSS:
		return rss.Parse(baseURL, r)
	case FormatJSON:
		return json.Parse(baseURL, r)
	case FormatRDF:
		return rdf.Parse(baseURL, r)
	default:
		return nil, ErrFeedFormatNotDetected
	}
}

func ParseBytes(urlStr string, b []byte) (feed *model.Feed, err error) {
	feedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("reader/parser: parse feed URL: %w", err)
	}

	switch gofeed.DetectFeedBytes(b) {
	case gofeed.FeedTypeAtom:
		feed, err = parseAtom(feedURL, b)
	case gofeed.FeedTypeRSS:
		feed, err = parseRSS(feedURL, b)
	case gofeed.FeedTypeJSON:
		feed, err = parseJSON(feedURL, b)
	default:
		return nil, ErrFeedFormatNotDetected
	}
	if err != nil {
		return nil, err
	}

	if feed.Title == "" {
		feed.Title = feed.SiteURL
	}
	return feed, nil
}
