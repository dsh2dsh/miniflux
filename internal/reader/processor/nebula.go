// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package processor // import "miniflux.app/v2/internal/reader/processor"

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
	"miniflux.app/v2/internal/reader/fetcher"
)

func shouldFetchNebulaWatchTime(entry *model.Entry) bool {
	if !config.Opts.FetchNebulaWatchTime() {
		return false
	}

	u, err := url.Parse(entry.URL)
	if err != nil {
		return false
	}

	return u.Hostname() == "nebula.tv"
}

func fetchNebulaWatchTime(websiteURL string) (int, error) {
	requestBuilder := fetcher.NewRequestBuilder()
	requestBuilder.WithTimeout(config.Opts.HTTPClientTimeout())
	requestBuilder.WithProxyRotator(proxyrotator.ProxyRotatorInstance)

	responseHandler := fetcher.NewResponseHandler(requestBuilder.ExecuteRequest(websiteURL))
	defer responseHandler.Close()

	if localizedError := responseHandler.LocalizedError(); localizedError != nil {
		slog.Warn("Unable to fetch Nebula watch time", slog.String("website_url", websiteURL), slog.Any("error", localizedError))
		return 0, localizedError
	}

	doc, docErr := goquery.NewDocumentFromReader(responseHandler.Body(config.Opts.HTTPClientMaxBodySize()))
	if docErr != nil {
		return 0, fmt.Errorf("reader/processor: %w", docErr)
	}

	durs, exists := doc.FindMatcher(goquery.Single(`meta[property="video:duration"]`)).Attr("content")
	// durs contains video watch time in seconds
	if !exists {
		return 0, errors.New("duration has not found")
	}

	dur, err := strconv.ParseInt(durs, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse duration %s: %w", durs, err)
	}

	return int(dur / 60), nil
}
