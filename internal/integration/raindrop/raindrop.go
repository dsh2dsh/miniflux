// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package raindrop // import "miniflux.app/v2/internal/integration/raindrop"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/reader/fetcher"
)

type Client struct {
	token        string
	collectionID string
	tags         []string
}

func NewClient(token, collectionID, tags string) *Client {
	return &Client{token: token, collectionID: collectionID, tags: strings.Split(tags, ",")}
}

// https://developer.raindrop.io/v1/raindrops/single#create-raindrop
func (c *Client) CreateRaindrop(ctx context.Context, entryURL,
	entryTitle string,
) error {
	if c.token == "" {
		return errors.New("raindrop: missing token")
	}

	var request *http.Request
	requestBodyJson, err := json.Marshal(&raindrop{
		Link:       entryURL,
		Title:      entryTitle,
		Collection: collection{Id: c.collectionID},
		Tags:       c.tags,
	})
	if err != nil {
		return fmt.Errorf("raindrop: unable to encode request body: %w", err)
	}

	request, err = http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.raindrop.io/rest/v1/raindrop",
		bytes.NewReader(requestBodyJson))
	if err != nil {
		return fmt.Errorf("raindrop: unable to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	request.Header.Set("Authorization", "Bearer "+c.token)

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("raindrop: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("raindrop: unable to create bookmark: status=%d",
			response.StatusCode())
	}
	return nil
}

type raindrop struct {
	Link       string     `json:"link"`
	Title      string     `json:"title"`
	Collection collection `json:"collection"`
	Tags       []string   `json:"tags"`
}

type collection struct {
	Id string `json:"$id"`
}
