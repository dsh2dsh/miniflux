// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archiveorg

import (
	"context"
	"fmt"
	"net/url"

	"miniflux.app/v2/internal/reader/fetcher"
)

// See https://docs.google.com/document/d/1Nsv52MvSjbLb2PCpHlat0gkzw0EvtSgpKHu4mk0MnrA/edit?tab=t.0
const options = "delay_wb_availability=1&if_not_archived_within=15d"

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) SendURL(ctx context.Context, entryURL string) error {
	requestURL := "https://web.archive.org/save/" + url.QueryEscape(entryURL) + "?" + options
	response, err := fetcher.Request(requestURL)
	if err != nil {
		return fmt.Errorf("archiveorg: unable to send request: %w", err)
	}
	defer response.Close()

	if response.StatusCode() >= 400 {
		return fmt.Errorf("archiveorg: unexpected status code: url=%s status=%d",
			requestURL, response.StatusCode())
	}
	return nil
}
