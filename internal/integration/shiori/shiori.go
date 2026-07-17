// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shiori // import "miniflux.app/v2/internal/integration/shiori"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/integration/client"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	baseURL  string
	username string
	password string
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{baseURL: baseURL, username: username, password: password}
}

func (c *Client) CreateBookmark(ctx context.Context, entryURL,
	entryTitle string,
) error {
	if c.baseURL == "" || c.username == "" || c.password == "" {
		return errors.New("shiori: missing base URL, username or password")
	}

	token, err := c.authenticate(ctx)
	if err != nil {
		return fmt.Errorf("shiori: unable to authenticate: %w", err)
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/bookmarks")
	if err != nil {
		return fmt.Errorf("shiori: invalid API endpoint: %w", err)
	}

	response, err := client.NewRequestBuilder(apiEndpoint).
		WithMethod(http.MethodPost).
		WithJSON(&addBookmarkRequest{
			URL:           entryURL,
			Title:         entryTitle,
			Excerpt:       "",
			CreateArchive: true,
			CreateEbook:   false,
			Public:        0,
			Tags:          make([]string, 0),
		}).
		WithHeader("Authorization", "Bearer "+token).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("shiori: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("shiori: unable to create bookmark: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

func (c *Client) authenticate(ctx context.Context) (string, error) {
	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/v1/auth/login")
	if err != nil {
		return "", fmt.Errorf("shiori: invalid API endpoint: %w", err)
	}

	response, err := client.NewRequestBuilder(apiEndpoint).
		WithMethod(http.MethodPost).
		WithJSON(&authRequest{Username: c.username, Password: c.password, RememberMe: false}).
		WithHeader("Accept", "application/json").
		Do(ctx)
	if err != nil {
		return "", fmt.Errorf("shiori: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("shiori: unable to authenticate: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}

	var authResponse authResponse
	if err := json.NewDecoder(response.Body()).Decode(&authResponse); err != nil {
		return "", fmt.Errorf("shiori: unable to decode response: %w", err)
	}
	return authResponse.Message.Token, nil
}

type authRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type authResponse struct {
	OK      bool                `json:"ok"`
	Message authResponseMessage `json:"message"`
}

type authResponseMessage struct {
	SessionID string `json:"session"`
	Token     string `json:"token"`
}

type addBookmarkRequest struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	CreateArchive bool     `json:"create_archive"`
	CreateEbook   bool     `json:"create_ebook"`
	Public        int      `json:"public"`
	Excerpt       string   `json:"excerpt"`
	Tags          []string `json:"tags"`
}
