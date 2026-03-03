// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package karakeep // import "miniflux.app/v2/internal/integration/karakeep"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"miniflux.app/v2/internal/reader/fetcher"
)

type Client struct {
	wrapped     *fetcher.RequestBuilder
	apiEndpoint string
	apiToken    string
	tags        string
}

type tagItem struct {
	TagName string `json:"tagName"`
}

type saveURLPayload struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type saveURLResponse struct {
	ID string `json:"id"`
}

type attachTagsPayload struct {
	Tags []tagItem `json:"tags"`
}

type errorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func NewClient(apiToken, apiEndpoint, tags string) *Client {
	return &Client{
		wrapped:     fetcher.NewRequestBuilder(),
		apiEndpoint: apiEndpoint,
		apiToken:    apiToken,
		tags:        tags,
	}
}

func (c *Client) attachTags(ctx context.Context, entryID string) error {
	if c.tags == "" {
		return nil
	}

	tagItems := make([]tagItem, 0)
	for tag := range strings.SplitSeq(c.tags, ",") {
		if trimmedTag := strings.TrimSpace(tag); trimmedTag != "" {
			tagItems = append(tagItems, tagItem{TagName: trimmedTag})
		}
	}

	if len(tagItems) == 0 {
		return nil
	}

	tagRequestBody, err := json.Marshal(&attachTagsPayload{
		Tags: tagItems,
	})
	if err != nil {
		return fmt.Errorf("karakeep: unable to encode tag request body: %w", err)
	}

	tagRequest, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/tags", c.apiEndpoint, entryID),
		bytes.NewReader(tagRequestBody))
	if err != nil {
		return fmt.Errorf("karakeep: unable to create tag request: %w", err)
	}

	tagRequest.Header.Set("Authorization", "Bearer "+c.apiToken)
	tagRequest.Header.Set("Content-Type", "application/json")

	tagResponse, err := c.wrapped.Do(tagRequest)
	if err != nil {
		return fmt.Errorf("karakeep: unable to send tag request: %w", err)
	}
	defer tagResponse.Close()

	if tagResponse.StatusCode() != http.StatusOK &&
		tagResponse.StatusCode() != http.StatusCreated {

		tagResponseBody, err := tagResponse.ReadBody()
		if err != nil {
			return fmt.Errorf("karakeep: failed to parse tag response: %w", err)
		}

		var errResponse errorResponse
		if err := json.Unmarshal(tagResponseBody, &errResponse); err != nil {
			return fmt.Errorf(
				"karakeep: unable to parse tag error response: status=%d body=%s",
				tagResponse.StatusCode(), string(tagResponseBody))
		}
		return fmt.Errorf(
			"karakeep: failed to attach tags: status=%d errorcode=%s %s",
			tagResponse.StatusCode(), errResponse.Code, errResponse.Error)
	}
	return nil
}

func (c *Client) SaveURL(ctx context.Context, entryURL string) error {
	b, err := json.Marshal(&saveURLPayload{Type: "link", URL: entryURL})
	if err != nil {
		return fmt.Errorf("integration/karakeep: unable to encode request body: %w",
			err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiEndpoint,
		bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("integration/karakeep: create POST request to %q: %w",
			c.apiEndpoint, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.wrapped.Do(req)
	if err != nil {
		return fmt.Errorf("integration/karakeep: unable to send request: %w", err)
	}
	defer resp.Close()

	b, err = io.ReadAll(resp.Body())
	if err != nil {
		return fmt.Errorf("integration/karakeep: failed to parse response: %w", err)
	}

	if resp.Header("Content-Type") != "application/json" {
		return fmt.Errorf(
			"integration/karakeep: unexpected content type response: %s",
			resp.Header("Content-Type"))
	}

	if resp.StatusCode() != http.StatusCreated {
		var errResponse errorResponse
		if err := json.Unmarshal(b, &errResponse); err != nil {
			return fmt.Errorf(
				"integration/karakeep: unable to parse error response: status=%d body=%s: %w",
				resp.StatusCode(), string(b), err)
		}
		return fmt.Errorf(
			"integration/karakeep: failed to save URL: status=%d errorcode=%s %s",
			resp.StatusCode(), errResponse.Code, errResponse.Error)
	}

	var response saveURLResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return fmt.Errorf("karakeep: unable to parse response: %w", err)
	}

	if response.ID == "" {
		return errors.New("karakeep: unable to get ID from response")
	}

	if err := c.attachTags(ctx, response.ID); err != nil {
		return fmt.Errorf("karakeep: unable to attach tags: %w", err)
	}
	return nil
}
