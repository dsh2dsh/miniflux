// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package betula // import "miniflux.app/v2/internal/integration/betula"

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	url   string
	token string
}

func NewClient(url, token string) *Client {
	return &Client{url: url, token: token}
}

func (c *Client) CreateBookmark(ctx context.Context, entryURL,
	entryTitle string, tags []string,
) error {
	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.url, "/save-link")
	if err != nil {
		return fmt.Errorf("betula: unable to generate save-link endpoint: %w", err)
	}

	values := url.Values{}
	values.Add("url", entryURL)
	values.Add("title", entryTitle)
	values.Add("tags", strings.Join(tags, ","))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		apiEndpoint+"?"+values.Encode(), nil)
	if err != nil {
		return fmt.Errorf("betula: unable to create request: %w", err)
	}

	request.AddCookie(&http.Cookie{Name: "betula-token", Value: c.token})
	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("betula: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("betula: unable to create bookmark: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}
