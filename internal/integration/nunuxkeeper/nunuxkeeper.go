// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nunuxkeeper // import "miniflux.app/v2/internal/integration/nunuxkeeper"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL string
	apiKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey}
}

func (c *Client) AddEntry(ctx context.Context, entryURL, entryTitle,
	entryContent string,
) error {
	if c.baseURL == "" || c.apiKey == "" {
		return errors.New("nunux-keeper: missing base URL or API key")
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/v2/documents")
	if err != nil {
		return fmt.Errorf(`nunux-keeper: invalid API endpoint: %w`, err)
	}

	requestBody, err := json.Marshal(&nunuxKeeperDocument{
		Title:       entryTitle,
		Origin:      entryURL,
		Content:     entryContent,
		ContentType: "text/html",
	})
	if err != nil {
		return fmt.Errorf("nunux-keeper: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint,
		bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("nunux-keeper: unable to create request: %w", err)
	}

	request.SetBasicAuth("api", c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("nunux-keeper: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf(
			"nunux-keeper: unable to create document: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

type nunuxKeeperDocument struct {
	Title       string `json:"title,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Content     string `json:"content,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}
