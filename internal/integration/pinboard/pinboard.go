// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package pinboard // import "miniflux.app/v2/internal/integration/pinboard"

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"miniflux.app/v2/internal/reader/fetcher"
)

var (
	errPostNotFound       = errors.New("pinboard: post not found")
	errMissingCredentials = errors.New("pinboard: missing auth token")
)

type Client struct {
	authToken string
}

func NewClient(authToken string) *Client {
	return &Client{authToken: authToken}
}

func (c *Client) CreateBookmark(ctx context.Context, entryURL, entryTitle,
	pinboardTags string, markAsUnread bool,
) error {
	if c.authToken == "" {
		return errMissingCredentials
	}

	// We check if the url is already bookmarked to avoid overriding existing data.
	post, err := c.getBookmark(ctx, entryURL)

	if err != nil && errors.Is(err, errPostNotFound) {
		post = NewPost(entryURL, entryTitle)
	} else if err != nil {
		// In case of any other error, we return immediately to avoid overriding existing data.
		return err
	}

	post.addTag(pinboardTags)
	if markAsUnread {
		post.SetToread()
	}

	values := url.Values{}
	values.Add("auth_token", c.authToken)
	post.AddValues(values)

	apiEndpoint := "https://api.pinboard.in/v1/posts/add?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint,
		nil)
	if err != nil {
		return fmt.Errorf("pinboard: unable to create request: %w", err)
	}

	response, err := fetcher.Do(request)
	if err != nil {
		return fmt.Errorf("pinboard: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf(
			"pinboard: unable to create a bookmark: url=%s status=%d",
			apiEndpoint, response.StatusCode())
	}
	return nil
}

// getBookmark fetches a bookmark from Pinboard. https://www.pinboard.in/api/#posts_get
func (c *Client) getBookmark(ctx context.Context, entryURL string) (*Post,
	error,
) {
	if c.authToken == "" {
		return nil, errMissingCredentials
	}

	values := url.Values{}
	values.Add("auth_token", c.authToken)
	values.Add("url", entryURL)

	apiEndpoint := "https://api.pinboard.in/v1/posts/get?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("pinboard: unable to create request: %w", err)
	}

	response, err := fetcher.Do(request)
	if err != nil {
		return nil, fmt.Errorf("pinboard: unable fetch bookmark: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return nil, fmt.Errorf("pinboard: unable to fetch bookmark, status=%d",
			response.StatusCode())
	}

	var results posts
	err = xml.NewDecoder(response.Body()).Decode(&results)
	if err != nil {
		return nil, fmt.Errorf("pinboard: unable to decode XML: %w", err)
	}

	if len(results.Posts) == 0 {
		return nil, errPostNotFound
	}
	return &results.Posts[0], nil
}
