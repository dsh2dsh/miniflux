// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shaarli // import "miniflux.app/v2/internal/integration/shaarli"

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL   string
	apiSecret string
}

func NewClient(baseURL, apiSecret string) *Client {
	return &Client{baseURL: baseURL, apiSecret: apiSecret}
}

func (c *Client) CreateLink(ctx context.Context, entryURL, entryTitle string,
) error {
	if c.baseURL == "" || c.apiSecret == "" {
		return errors.New("shaarli: missing base URL or API secret")
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/v1/links")
	if err != nil {
		return fmt.Errorf("shaarli: invalid API endpoint: %w", err)
	}

	requestBody, err := json.Marshal(&addLinkRequest{
		URL:     entryURL,
		Title:   entryTitle,
		Private: true,
	})
	if err != nil {
		return fmt.Errorf("shaarli: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint,
		bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("shaarli: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.generateBearerToken())

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("shaarli: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusCreated {
		return fmt.Errorf("shaarli: unable to add link: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

func (c *Client) generateBearerToken() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"HS512"}`))
	payload := base64.RawURLEncoding.EncodeToString(
		fmt.Appendf(nil, `{"iat":%d}`, time.Now().Unix()))
	data := header + "." + payload

	mac := hmac.New(sha512.New, []byte(c.apiSecret))
	mac.Write([]byte(data))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return data + "." + signature
}

type addLinkRequest struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Private bool   `json:"private"`
}
