// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package karakeep // import "miniflux.app/v2/internal/integration/karakeep"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"miniflux.app/v2/internal/version"
)

const defaultClientTimeout = 10 * time.Second

type errorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

type saveURLPayload struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Client struct {
	wrapped     *http.Client
	apiEndpoint string
	apiToken    string
}

func NewClient(apiToken, apiEndpoint string) *Client {
	return &Client{
		wrapped:     &http.Client{Timeout: defaultClientTimeout},
		apiEndpoint: apiEndpoint,
		apiToken:    apiToken,
	}
}

func (c *Client) SaveURL(entryURL string) error {
	b, err := json.Marshal(&saveURLPayload{Type: "link", URL: entryURL})
	if err != nil {
		return fmt.Errorf("integration/karakeep: unable to encode request body: %w",
			err)
	}

	req, err := http.NewRequest(http.MethodPost, c.apiEndpoint,
		bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("integration/karakeep: create POST request to %q: %w",
			c.apiEndpoint, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Miniflux/"+version.Version)

	resp, err := c.wrapped.Do(req)
	if err != nil {
		return fmt.Errorf("integration/karakeep: unable to send request: %w", err)
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("integration/karakeep: failed to parse response: %w", err)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf(
			"integration/karakeep: unexpected content type response: %s",
			resp.Header.Get("Content-Type"))
	}

	if resp.StatusCode != http.StatusCreated {
		var errResponse errorResponse
		if err := json.Unmarshal(b, &errResponse); err != nil {
			return fmt.Errorf(
				"integration/karakeep: unable to parse error response: status=%d body=%s: %w",
				resp.StatusCode, string(b), err)
		}
		return fmt.Errorf(
			"integration/karakeep: failed to save URL: status=%d errorcode=%s %s",
			resp.StatusCode, errResponse.Code, errResponse.Error)
	}
	return nil
}
