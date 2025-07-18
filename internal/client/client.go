// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package client // import "miniflux.app/v2/client"

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"miniflux.app/v2/internal/model"
)

// Client holds API procedure calls.
type Client struct {
	request *request
}

// New returns a new Miniflux client.
//
// Deprecated: use NewClient instead.
func New(endpoint string, credentials ...string) *Client {
	return NewClient(endpoint, credentials...)
}

// NewClient returns a new Miniflux client.
func NewClient(endpoint string, credentials ...string) *Client {
	// Trim trailing slashes and /v1 from the endpoint.
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = strings.TrimSuffix(endpoint, "/v1")
	switch len(credentials) {
	case 2:
		return &Client{request: &request{endpoint: endpoint, username: credentials[0], password: credentials[1]}}
	case 1:
		return &Client{request: &request{endpoint: endpoint, apiKey: credentials[0]}}
	default:
		return &Client{request: &request{endpoint: endpoint}}
	}
}

// Healthcheck checks if the application is up and running.
func (c *Client) Healthcheck() error {
	body, err := c.request.Get("/healthcheck")
	if err != nil {
		return fmt.Errorf("miniflux: unable to perform healthcheck: %w", err)
	}
	defer body.Close()

	responseBodyContent, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("miniflux: unable to read healthcheck response: %w", err)
	}

	if string(responseBodyContent) != "OK" {
		return fmt.Errorf("miniflux: invalid healthcheck response: %q", responseBodyContent)
	}

	return nil
}

// Version returns the version of the Miniflux instance.
func (c *Client) Version() (*VersionResponse, error) {
	body, err := c.request.Get("/v1/version")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var versionResponse *VersionResponse
	if err := json.NewDecoder(body).Decode(&versionResponse); err != nil {
		return nil, fmt.Errorf("miniflux: json error (%w)", err)
	}

	return versionResponse, nil
}

// Me returns the logged user information.
func (c *Client) Me() (*model.User, error) {
	body, err := c.request.Get("/v1/me")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var user *model.User
	if err := json.NewDecoder(body).Decode(&user); err != nil {
		return nil, fmt.Errorf("miniflux: json error (%w)", err)
	}

	return user, nil
}

// Users returns all users.
func (c *Client) Users() (model.Users, error) {
	body, err := c.request.Get("/v1/users")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var users model.Users
	if err := json.NewDecoder(body).Decode(&users); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return users, nil
}

// UserByID returns a single user.
func (c *Client) UserByID(userID int64) (*model.User, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/users/%d", userID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var user model.User
	if err := json.NewDecoder(body).Decode(&user); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &user, nil
}

// UserByUsername returns a single user.
func (c *Client) UserByUsername(username string) (*model.User, error) {
	body, err := c.request.Get("/v1/users/" + username)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var user model.User
	if err := json.NewDecoder(body).Decode(&user); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &user, nil
}

// CreateUser creates a new user in the system.
func (c *Client) CreateUser(username, password string, isAdmin bool) (*model.User, error) {
	body, err := c.request.Post("/v1/users", &model.UserCreationRequest{
		Username: username,
		Password: password,
		IsAdmin:  isAdmin,
	})
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var user *model.User
	if err := json.NewDecoder(body).Decode(&user); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return user, nil
}

// UpdateUser updates a user in the system.
func (c *Client) UpdateUser(userID int64, userChanges *model.UserModificationRequest) (*model.User, error) {
	body, err := c.request.Put(fmt.Sprintf("/v1/users/%d", userID), userChanges)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var u *model.User
	if err := json.NewDecoder(body).Decode(&u); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return u, nil
}

// DeleteUser removes a user from the system.
func (c *Client) DeleteUser(userID int64) error {
	return c.request.Delete(fmt.Sprintf("/v1/users/%d", userID))
}

// APIKeys returns all API keys for the authenticated user.
func (c *Client) APIKeys() ([]*model.APIKey, error) {
	body, err := c.request.Get("/v1/api-keys")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var apiKeys []*model.APIKey
	if err := json.NewDecoder(body).Decode(&apiKeys); err != nil {
		return nil, fmt.Errorf("miniflux: response error: %w", err)
	}

	return apiKeys, nil
}

// CreateAPIKey creates a new API key for the authenticated user.
func (c *Client) CreateAPIKey(description string) (*model.APIKey, error) {
	body, err := c.request.Post("/v1/api-keys", &model.APIKeyCreationRequest{
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var apiKey *model.APIKey
	if err := json.NewDecoder(body).Decode(&apiKey); err != nil {
		return nil, fmt.Errorf("miniflux: response error: %w", err)
	}

	return apiKey, nil
}

// DeleteAPIKey removes an API key for the authenticated user.
func (c *Client) DeleteAPIKey(apiKeyID int64) error {
	return c.request.Delete(fmt.Sprintf("/v1/api-keys/%d", apiKeyID))
}

// MarkAllAsRead marks all unread entries as read for a given user.
func (c *Client) MarkAllAsRead(userID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/users/%d/mark-all-as-read", userID), nil)
	return err
}

// IntegrationsStatus fetches the integrations status for the logged user.
func (c *Client) IntegrationsStatus() (bool, error) {
	body, err := c.request.Get("/v1/integrations/status")
	if err != nil {
		return false, err
	}
	defer body.Close()

	var response struct {
		HasIntegrations bool `json:"has_integrations"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return false, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return response.HasIntegrations, nil
}

// Discover try to find subscriptions from a website.
func (c *Client) Discover(url string) (Subscriptions, error) {
	body, err := c.request.Post("/v1/discover", map[string]string{"url": url})
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var subscriptions Subscriptions
	if err := json.NewDecoder(body).Decode(&subscriptions); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return subscriptions, nil
}

// Categories gets the list of categories.
func (c *Client) Categories() ([]*model.Category, error) {
	body, err := c.request.Get("/v1/categories")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var categories []*model.Category
	if err := json.NewDecoder(body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return categories, nil
}

// CreateCategory creates a new category.
func (c *Client) CreateCategory(title string) (*model.Category, error) {
	body, err := c.request.Post("/v1/categories", &model.CategoryCreationRequest{
		Title: title,
	})
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var category *model.Category
	if err := json.NewDecoder(body).Decode(&category); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return category, nil
}

// CreateCategoryWithOptions creates a new category with options.
func (c *Client) CreateCategoryWithOptions(createRequest *model.CategoryCreationRequest) (*model.Category, error) {
	body, err := c.request.Post("/v1/categories", createRequest)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var category *model.Category
	if err := json.NewDecoder(body).Decode(&category); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}
	return category, nil
}

// UpdateCategory updates a category.
func (c *Client) UpdateCategory(categoryID int64, title string) (*model.Category, error) {
	body, err := c.request.Put(fmt.Sprintf("/v1/categories/%d", categoryID), &model.CategoryModificationRequest{
		Title: model.SetOptionalField(title),
	})
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var category *model.Category
	if err := json.NewDecoder(body).Decode(&category); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return category, nil
}

// UpdateCategoryWithOptions updates a category with options.
func (c *Client) UpdateCategoryWithOptions(categoryID int64, categoryChanges *model.CategoryModificationRequest) (*model.Category, error) {
	body, err := c.request.Put(fmt.Sprintf("/v1/categories/%d", categoryID), categoryChanges)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var category *model.Category
	if err := json.NewDecoder(body).Decode(&category); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return category, nil
}

// MarkCategoryAsRead marks all unread entries in a category as read.
func (c *Client) MarkCategoryAsRead(categoryID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/categories/%d/mark-all-as-read", categoryID), nil)
	return err
}

// CategoryFeeds gets feeds of a category.
func (c *Client) CategoryFeeds(categoryID int64) (model.Feeds, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/categories/%d/feeds", categoryID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var feeds model.Feeds
	if err := json.NewDecoder(body).Decode(&feeds); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return feeds, nil
}

// DeleteCategory removes a category.
func (c *Client) DeleteCategory(categoryID int64) error {
	return c.request.Delete(fmt.Sprintf("/v1/categories/%d", categoryID))
}

// RefreshCategory refreshes a category.
func (c *Client) RefreshCategory(categoryID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/categories/%d/refresh", categoryID), nil)
	return err
}

// Feeds gets all feeds.
func (c *Client) Feeds() (model.Feeds, error) {
	body, err := c.request.Get("/v1/feeds")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var feeds model.Feeds
	if err := json.NewDecoder(body).Decode(&feeds); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return feeds, nil
}

// Export creates OPML file.
func (c *Client) Export() ([]byte, error) {
	body, err := c.request.Get("/v1/export")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	opml, err := io.ReadAll(body)
	if err != nil {
		return nil, err //nolint:wrapcheck // no reason
	}

	return opml, nil
}

// Import imports an OPML file.
func (c *Client) Import(f io.ReadCloser) error {
	_, err := c.request.PostFile("/v1/import", f)
	return err
}

// Feed gets a feed.
func (c *Client) Feed(feedID int64) (*model.Feed, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/feeds/%d", feedID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var feed *model.Feed
	if err := json.NewDecoder(body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return feed, nil
}

// CreateFeed creates a new feed.
func (c *Client) CreateFeed(feedCreationRequest *model.FeedCreationRequest) (int64, error) {
	body, err := c.request.Post("/v1/feeds", feedCreationRequest)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	type result struct {
		FeedID int64 `json:"feed_id"`
	}

	var r result
	if err := json.NewDecoder(body).Decode(&r); err != nil {
		return 0, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return r.FeedID, nil
}

// UpdateFeed updates a feed.
func (c *Client) UpdateFeed(feedID int64, feedChanges *model.FeedModificationRequest) (*model.Feed, error) {
	body, err := c.request.Put(fmt.Sprintf("/v1/feeds/%d", feedID), feedChanges)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var f *model.Feed
	if err := json.NewDecoder(body).Decode(&f); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return f, nil
}

// MarkFeedAsRead marks all unread entries of the feed as read.
func (c *Client) MarkFeedAsRead(feedID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/feeds/%d/mark-all-as-read", feedID), nil)
	return err
}

// RefreshAllFeeds refreshes all feeds.
func (c *Client) RefreshAllFeeds() error {
	_, err := c.request.Put("/v1/feeds/refresh", nil)
	return err
}

// RefreshFeed refreshes a feed.
func (c *Client) RefreshFeed(feedID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/feeds/%d/refresh", feedID), nil)
	return err
}

// DeleteFeed removes a feed.
func (c *Client) DeleteFeed(feedID int64) error {
	return c.request.Delete(fmt.Sprintf("/v1/feeds/%d", feedID))
}

// FeedIcon gets a feed icon.
func (c *Client) FeedIcon(feedID int64) (*FeedIcon, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/feeds/%d/icon", feedID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var feedIcon *FeedIcon
	if err := json.NewDecoder(body).Decode(&feedIcon); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return feedIcon, nil
}

// FeedEntry gets a single feed entry.
func (c *Client) FeedEntry(feedID, entryID int64) (*model.Entry, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/feeds/%d/entries/%d", feedID, entryID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var entry *model.Entry
	if err := json.NewDecoder(body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return entry, nil
}

// CategoryEntry gets a single category entry.
func (c *Client) CategoryEntry(categoryID, entryID int64) (*model.Entry, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/categories/%d/entries/%d", categoryID, entryID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var entry *model.Entry
	if err := json.NewDecoder(body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return entry, nil
}

// Entry gets a single entry.
func (c *Client) Entry(entryID int64) (*model.Entry, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/entries/%d", entryID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var entry *model.Entry
	if err := json.NewDecoder(body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return entry, nil
}

// Entries fetch entries.
func (c *Client) Entries(filter *Filter) (*EntryResultSet, error) {
	path := buildFilterQueryString("/v1/entries", filter)

	body, err := c.request.Get(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var result EntryResultSet
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &result, nil
}

// FeedEntries fetch feed entries.
func (c *Client) FeedEntries(feedID int64, filter *Filter) (*EntryResultSet, error) {
	path := buildFilterQueryString(fmt.Sprintf("/v1/feeds/%d/entries", feedID), filter)

	body, err := c.request.Get(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var result EntryResultSet
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &result, nil
}

// CategoryEntries fetch entries of a category.
func (c *Client) CategoryEntries(categoryID int64, filter *Filter) (*EntryResultSet, error) {
	path := buildFilterQueryString(fmt.Sprintf("/v1/categories/%d/entries", categoryID), filter)

	body, err := c.request.Get(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var result EntryResultSet
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &result, nil
}

// UpdateEntries updates the status of a list of entries.
func (c *Client) UpdateEntries(entryIDs []int64, status string) error {
	type payload struct {
		EntryIDs []int64 `json:"entry_ids"`
		Status   string  `json:"status"`
	}

	_, err := c.request.Put("/v1/entries", &payload{EntryIDs: entryIDs, Status: status})
	return err
}

// UpdateEntry updates an entry.
func (c *Client) UpdateEntry(entryID int64, entryChanges *model.EntryUpdateRequest) (*model.Entry, error) {
	body, err := c.request.Put(fmt.Sprintf("/v1/entries/%d", entryID), entryChanges)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var entry *model.Entry
	if err := json.NewDecoder(body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return entry, nil
}

// ToggleBookmark toggles entry bookmark value.
func (c *Client) ToggleBookmark(entryID int64) error {
	_, err := c.request.Put(fmt.Sprintf("/v1/entries/%d/bookmark", entryID), nil)
	return err
}

// SaveEntry sends an entry to a third-party service.
func (c *Client) SaveEntry(entryID int64) error {
	_, err := c.request.Post(fmt.Sprintf("/v1/entries/%d/save", entryID), nil)
	return err
}

// FetchEntryOriginalContent fetches the original content of an entry using the scraper.
func (c *Client) FetchEntryOriginalContent(entryID int64) (string, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/entries/%d/fetch-content", entryID))
	if err != nil {
		return "", err
	}
	defer body.Close()

	var response struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return "", fmt.Errorf("miniflux: response error (%w)", err)
	}

	return response.Content, nil
}

// FetchCounters fetches feed counters.
func (c *Client) FetchCounters() (*model.FeedCounters, error) {
	body, err := c.request.Get("/v1/feeds/counters")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var result model.FeedCounters
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return &result, nil
}

// FlushHistory changes all entries with the status "read" to "removed".
func (c *Client) FlushHistory() error {
	_, err := c.request.Put("/v1/flush-history", nil)
	return err
}

// Icon fetches a feed icon.
func (c *Client) Icon(iconID int64) (*FeedIcon, error) {
	body, err := c.request.Get(fmt.Sprintf("/v1/icons/%d", iconID))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var feedIcon *FeedIcon
	if err := json.NewDecoder(body).Decode(&feedIcon); err != nil {
		return nil, fmt.Errorf("miniflux: response error (%w)", err)
	}

	return feedIcon, nil
}

// UpdateEnclosure updates an enclosure.
func (c *Client) UpdateEnclosure(entryID, at int64,
	data *model.EnclosureUpdateRequest,
) error {
	_, err := c.request.Put(
		fmt.Sprintf("/v1/entries/%d/enclosure/%d", entryID, at), data)
	return err
}

func buildFilterQueryString(path string, filter *Filter) string {
	if filter != nil {
		values := url.Values{}

		if filter.Status != "" {
			values.Set("status", filter.Status)
		}

		if filter.Direction != "" {
			values.Set("direction", filter.Direction)
		}

		if filter.Order != "" {
			values.Set("order", filter.Order)
		}

		if filter.Limit >= 0 {
			values.Set("limit", strconv.Itoa(filter.Limit))
		}

		if filter.Offset >= 0 {
			values.Set("offset", strconv.Itoa(filter.Offset))
		}

		if filter.After > 0 {
			values.Set("after", strconv.FormatInt(filter.After, 10))
		}

		if filter.Before > 0 {
			values.Set("before", strconv.FormatInt(filter.Before, 10))
		}

		if filter.PublishedAfter > 0 {
			values.Set("published_after", strconv.FormatInt(filter.PublishedAfter, 10))
		}

		if filter.PublishedBefore > 0 {
			values.Set("published_before", strconv.FormatInt(filter.PublishedBefore, 10))
		}

		if filter.ChangedAfter > 0 {
			values.Set("changed_after", strconv.FormatInt(filter.ChangedAfter, 10))
		}

		if filter.ChangedBefore > 0 {
			values.Set("changed_before", strconv.FormatInt(filter.ChangedBefore, 10))
		}

		if filter.AfterEntryID > 0 {
			values.Set("after_entry_id", strconv.FormatInt(filter.AfterEntryID, 10))
		}

		if filter.BeforeEntryID > 0 {
			values.Set("before_entry_id", strconv.FormatInt(filter.BeforeEntryID, 10))
		}

		if filter.Starred != "" {
			values.Set("starred", filter.Starred)
		}

		if filter.Search != "" {
			values.Set("search", filter.Search)
		}

		if filter.CategoryID > 0 {
			values.Set("category_id", strconv.FormatInt(filter.CategoryID, 10))
		}

		if filter.FeedID > 0 {
			values.Set("feed_id", strconv.FormatInt(filter.FeedID, 10))
		}

		if filter.GloballyVisible {
			values.Set("globally_visible", "true")
		}

		for _, status := range filter.Statuses {
			values.Add("status", status)
		}

		path = fmt.Sprintf("%s?%s", path, values.Encode())
	}

	return path
}
