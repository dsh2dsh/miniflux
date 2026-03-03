// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package instapaper // import "miniflux.app/v2/internal/integration/instapaper"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"miniflux.app/v2/internal/reader/fetcher"
)

type Client struct {
	username string
	password string
}

func NewClient(username, password string) *Client {
	return &Client{username: username, password: password}
}

func (c *Client) AddURL(ctx context.Context, entryURL, entryTitle string) error {
	if c.username == "" || c.password == "" {
		return errors.New("instapaper: missing username or password")
	}

	values := url.Values{}
	values.Add("url", entryURL)
	values.Add("title", entryTitle)

	apiEndpoint := "https://www.instapaper.com/api/add?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint,
		nil)
	if err != nil {
		return fmt.Errorf("instapaper: unable to create request: %w", err)
	}

	request.SetBasicAuth(c.username, c.password)
	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("instapaper: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusCreated {
		return fmt.Errorf("instapaper: unable to add URL: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}
