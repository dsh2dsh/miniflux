// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"miniflux.app/v2/internal/reader/fetcher"
)

type Client struct {
	apiToken string
	pageID   string
}

func NewClient(apiToken, pageID string) *Client {
	return &Client{apiToken, pageID}
}

func (c *Client) UpdateDocument(ctx context.Context, entryURL,
	entryTitle string,
) error {
	if c.apiToken == "" || c.pageID == "" {
		return errors.New("notion: missing API token or page ID")
	}

	apiEndpoint := "https://api.notion.com/v1/blocks/" + c.pageID + "/children"
	requestBody, err := json.Marshal(&notionDocument{
		Children: []block{
			{
				Object: "block",
				Type:   "bookmark",
				Bookmark: bookmarkObject{
					Caption: []any{},
					URL:     entryURL,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("notion: unable to encode request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, apiEndpoint,
		bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("notion: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Notion-Version", "2022-06-28")
	request.Header.Set("Authorization", "Bearer "+c.apiToken)

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("notion: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("notion: unable to update document: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

type notionDocument struct {
	Children []block `json:"children"`
}

type block struct {
	Object   string         `json:"object"`
	Type     string         `json:"type"`
	Bookmark bookmarkObject `json:"bookmark"`
}

type bookmarkObject struct {
	Caption []any  `json:"caption"`
	URL     string `json:"url"`
}
