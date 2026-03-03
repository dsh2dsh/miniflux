// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package linkding // import "miniflux.app/v2/internal/integration/linkding"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL string
	apiKey  string
	tags    string
	unread  bool
}

func NewClient(baseURL, apiKey, tags string, unread bool) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, tags: tags, unread: unread}
}

func (c *Client) CreateBookmark(ctx context.Context, entryURL,
	entryTitle string,
) error {
	if c.baseURL == "" || c.apiKey == "" {
		return errors.New("linkding: missing base URL or API key")
	}

	tagsSplitFn := func(c rune) bool {
		return c == ',' || c == ' '
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/bookmarks/")
	if err != nil {
		return fmt.Errorf(`linkding: invalid API endpoint: %w`, err)
	}

	requestBody, err := json.Marshal(&linkdingBookmark{
		URL:      entryURL,
		Title:    entryTitle,
		TagNames: strings.FieldsFunc(c.tags, tagsSplitFn),
		Unread:   c.unread,
	})
	if err != nil {
		return fmt.Errorf("linkding: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint,
		bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("linkding: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Token "+c.apiKey)

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("linkding: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("linkding: unable to create bookmark: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

type linkdingBookmark struct {
	URL      string   `json:"url,omitempty"`
	Title    string   `json:"title,omitempty"`
	TagNames []string `json:"tag_names,omitempty"`
	Unread   bool     `json:"unread,omitempty"`
}
