// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/urllib"
)

type Client struct {
	servicesURL string
	baseURL     string
}

func NewClient(serviceURL, baseURL string) *Client {
	return &Client{servicesURL: serviceURL, baseURL: baseURL}
}

func (c *Client) SendNotification(ctx context.Context, feed *model.Feed,
	entries model.Entries,
) error {
	if c.baseURL == "" || c.servicesURL == "" {
		return errors.New("apprise: missing base URL or services URL")
	}

	for _, entry := range entries {
		message := "[" + entry.Title + "]" + "(" + entry.URL + ")" + "\n\n"
		apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/notify")
		if err != nil {
			return fmt.Errorf(`apprise: invalid API endpoint: %w`, err)
		}

		requestBody, err := json.Marshal(map[string]any{
			"urls":  c.servicesURL,
			"body":  message,
			"title": feed.Title,
		})
		if err != nil {
			return fmt.Errorf("apprise: unable to encode request body: %w", err)
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost,
			apiEndpoint, bytes.NewReader(requestBody))
		if err != nil {
			return fmt.Errorf("apprise: unable to create request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")

		slog.Debug("Sending Apprise notification",
			slog.String("apprise_url", c.baseURL),
			slog.String("services_url", c.servicesURL),
			slog.String("title", feed.Title),
			slog.String("body", message),
			slog.String("entry_url", entry.URL),
		)

		response, err := fetcher.Do(request)
		if err != nil {
			return fmt.Errorf("apprise: unable to send request: %w", err)
		}
		defer response.Close()

		if response.StatusCode() >= 400 {
			return fmt.Errorf(
				"apprise: unable to send a notification: url=%s status=%d",
				apiEndpoint, response.StatusCode())
		}
	}
	return nil
}
