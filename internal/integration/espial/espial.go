// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package espial // import "miniflux.app/v2/internal/integration/espial"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/integration/client"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL string
	apiKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey}
}

func (c *Client) CreateLink(ctx context.Context, entryURL, entryTitle,
	espialTags string,
) error {
	if c.baseURL == "" || c.apiKey == "" {
		return errors.New("espial: missing base URL or API key")
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/add")
	if err != nil {
		return fmt.Errorf("espial: invalid API endpoint: %w", err)
	}

	response, err := client.NewRequestBuilder(apiEndpoint).
		WithMethod(http.MethodPost).
		WithJSON(&espialDocument{
			Title:  entryTitle,
			URL:    entryURL,
			ToRead: true,
			Tags:   espialTags,
		}).
		WithHeader("Authorization", "ApiKey "+c.apiKey).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("espial: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusCreated {
		var responseBody bytes.Buffer
		_ = response.WriteBodyTo(&responseBody)
		return fmt.Errorf("espial: unable to create link: url=%s status=%d body=%s",
			apiEndpoint, response.StatusCode(), responseBody.String())
	}
	return nil
}

type espialDocument struct {
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
	ToRead bool   `json:"toread,omitempty"`
	Tags   string `json:"tags,omitempty"`
}
