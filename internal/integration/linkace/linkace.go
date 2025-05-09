package linkace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"miniflux.app/v2/internal/urllib"
	"miniflux.app/v2/internal/version"
)

const defaultClientTimeout = 10 * time.Second

type Client struct {
	baseURL       string
	apiKey        string
	tags          string
	private       bool
	checkDisabled bool
}

func NewClient(baseURL, apiKey, tags string, private bool, checkDisabled bool) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, tags: tags, private: private, checkDisabled: checkDisabled}
}

func (c *Client) AddURL(entryURL, entryTitle string) error {
	if c.baseURL == "" || c.apiKey == "" {
		return errors.New("linkace: missing base URL or API key")
	}

	tagsSplitFn := func(c rune) bool {
		return c == ',' || c == ' '
	}

	apiEndpoint, err := urllib.JoinBaseURLAndPath(c.baseURL, "/api/v2/links")
	if err != nil {
		return fmt.Errorf("linkace: invalid API endpoint: %w", err)
	}
	requestBody, err := json.Marshal(&createItemRequest{
		Url:           entryURL,
		Title:         entryTitle,
		Tags:          strings.FieldsFunc(c.tags, tagsSplitFn),
		Private:       c.private,
		CheckDisabled: c.checkDisabled,
	})
	if err != nil {
		return fmt.Errorf("linkace: unable to encode request body: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, apiEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("linkace: unable to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Miniflux/"+version.Version)
	request.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpClient := &http.Client{Timeout: defaultClientTimeout}
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("linkace: unable to send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return fmt.Errorf("linkace: unable to create item: url=%s status=%d", apiEndpoint, response.StatusCode)
	}

	return nil
}

type createItemRequest struct {
	Title         string   `json:"title,omitempty"`
	Url           string   `json:"url"`
	Tags          []string `json:"tags,omitempty"`
	Private       bool     `json:"is_private,omitempty"`
	CheckDisabled bool     `json:"check_disabled,omitempty"`
}
