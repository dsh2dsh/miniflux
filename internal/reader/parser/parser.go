// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package parser // import "miniflux.app/v2/internal/reader/parser"

import (
	"errors"
	"fmt"
	"io"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/atom"
	"miniflux.app/v2/internal/reader/json"
	"miniflux.app/v2/internal/reader/rdf"
	"miniflux.app/v2/internal/reader/rss"
)

var ErrFeedFormatNotDetected = errors.New("parser: unable to detect feed format")

// ParseFeed analyzes the input data and returns a normalized feed object.
func ParseFeed(baseURL string, r io.ReadSeeker) (*model.Feed, error) {
	_, err := r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("reader/parser: failed rewind to begin: %w", err)
	}

	format, version := DetectFeedFormat(r)
	switch format {
	case FormatAtom:
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			break
		}
		return atom.Parse(baseURL, r, version)
	case FormatRSS:
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			break
		}
		return rss.Parse(baseURL, r)
	case FormatJSON:
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			break
		}
		return json.Parse(baseURL, r)
	case FormatRDF:
		if _, err = r.Seek(0, io.SeekStart); err != nil {
			break
		}
		return rdf.Parse(baseURL, r)
	default:
		return nil, ErrFeedFormatNotDetected
	}
	if err != nil {
		return nil, fmt.Errorf("reader/parser: failed rewind to begin: %w", err)
	}
	return nil, nil
}
