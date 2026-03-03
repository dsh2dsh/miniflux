// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package wallabag // import "miniflux.app/v2/internal/integration/wallabag"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	username     string
	password     string
	tags         string
	onlyURL      bool
}

func NewClient(baseURL, clientID, clientSecret, username, password, tags string, onlyURL bool) *Client {
	return &Client{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		tags:         tags,
		onlyURL:      onlyURL,
	}
}

func (c *Client) CreateEntry(ctx context.Context, entryURL, entryTitle,
	entryContent string,
) error {
	if c.baseURL == "" || c.clientID == "" || c.clientSecret == "" || c.username == "" || c.password == "" {
		return errors.New("wallabag: missing base URL, client ID, client secret, username or password")
	}

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	return c.createEntry(ctx, accessToken, entryURL, entryTitle, entryContent,
		c.tags)
}

func (c *Client) createEntry(ctx context.Context, accessToken, entryURL,
	entryTitle, entryContent, tags string,
) error {
	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/entries.json")
	if err != nil {
		return fmt.Errorf("wallabag: unable to generate entries endpoint: %w", err)
	}

	if c.onlyURL {
		entryContent = ""
	}

	requestBody, err := json.Marshal(&createEntryRequest{
		URL:     entryURL,
		Title:   entryTitle,
		Content: entryContent,
		Tags:    tags,
	})
	if err != nil {
		return fmt.Errorf("wallabag: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint,
		bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("wallabag: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("wallabag: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("wallabag: unable to get save entry: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	values := url.Values{}
	values.Add("grant_type", "password")
	values.Add("client_id", c.clientID)
	values.Add("client_secret", c.clientSecret)
	values.Add("username", c.username)
	values.Add("password", c.password)

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/oauth/v2/token")
	if err != nil {
		return "", fmt.Errorf("wallabag: unable to generate token endpoint: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint,
		strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("wallabag: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := fetcher.Do(request)
	if err != nil {
		return "", fmt.Errorf("wallabag: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return "", fmt.Errorf(
			"wallabag: unable to get access token: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}

	var responseBody tokenResponse
	if err := json.NewDecoder(response.Body()).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("wallabag: unable to decode token response: %w", err)
	}
	return responseBody.AccessToken, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	Expires      int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

type createEntryRequest struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Tags    string `json:"tags,omitempty"`
}
