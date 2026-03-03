// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Readwise Reader API documentation: https://readwise.io/reader_api

package readwise // import "miniflux.app/v2/internal/integration/readwise"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/reader/fetcher"
)

const readwiseApiEndpoint = "https://readwise.io/api/v3/save/"

type Client struct {
	apiKey string
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) CreateDocument(ctx context.Context, entryURL string) error {
	if c.apiKey == "" {
		return errors.New("readwise: missing API key")
	}

	requestBody, err := json.Marshal(&readwiseDocument{
		URL: entryURL,
	})
	if err != nil {
		return fmt.Errorf("readwise: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		readwiseApiEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("readwise: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Token "+c.apiKey)

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("readwise: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("readwise: unable to create document: url=%s status=%d",
			readwiseApiEndpoint, response.StatusCode())
	}
	return nil
}

type readwiseDocument struct {
	URL string `json:"url"`
}
