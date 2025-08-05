// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//go:build e2e

package api // import "miniflux.app/v2/internal/api"

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/caarlos0/env/v11"
	dotenv "github.com/dsh2dsh/expx-dotenv"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"miniflux.app/v2/internal/client"
	"miniflux.app/v2/internal/model"
)

func TestHealthcheckEndpoint(t *testing.T) {
	cfg := NewIntegrationConfig(t)
	api := client.NewClient(cfg.BaseURL)
	require.NoError(t, api.Healthcheck())
}

func NewIntegrationConfig(t *testing.T) *IntegrationConfig {
	t.Helper()

	c := &IntegrationConfig{
		RegularUsername:   "regular_test_user",
		RegularPassword:   "regular_test_user_password",
		FeedURL:           "http://127.0.0.1:8000/feed.xml",
		FeedTitle:         "Miniflux",
		SubscriptionTitle: "Miniflux Releases",
		WebsiteURL:        "http://127.0.0.1:8000",
		TestListenAddr:    "127.0.0.1:8000",
	}

	err := dotenv.New().Load(func() error { return env.Parse(c) })
	require.NoError(t, err)
	return c
}

type IntegrationConfig struct {
	BaseURL           string `env:"TEST_MINIFLUX_BASE_URL,required"`
	AdminUsername     string `env:"ADMIN_USERNAME,required"`
	AdminPassword     string `env:"ADMIN_PASSWORD,required"`
	RegularUsername   string `env:"TEST_MINIFLUX_REGULAR_USERNAME_PREFIX"`
	RegularPassword   string `env:"TEST_MINIFLUX_REGULAR_PASSWORD"`
	FeedURL           string `env:"TEST_MINIFLUX_FEED_URL"`
	FeedTitle         string `env:"TEST_MINIFLUX_FEED_TITLE"`
	SubscriptionTitle string `env:"TEST_MINIFLUX_SUBSCRIPTION_TITLE"`
	WebsiteURL        string `env:"TEST_MINIFLUX_WEBSITE_URL"`
	TestListenAddr    string `env:"TEST_LISTEN_ADDR"`
}

func (c *IntegrationConfig) RandomUsername() string {
	return fmt.Sprintf("%s_%10d", c.RegularUsername, rand.Int())
}

func TestEndpointSuite(t *testing.T) {
	cfg := NewIntegrationConfig(t)
	admin := client.NewClient(cfg.BaseURL, cfg.AdminUsername,
		cfg.AdminPassword)
	require.NotNil(t, admin)
	require.NoError(t, admin.Healthcheck())

	l, err := net.Listen("tcp", cfg.TestListenAddr)
	require.NoError(t, err)
	ts := httptest.NewUnstartedServer(http.FileServer(http.Dir("testdata")))
	ts.Listener = l
	ts.Start()
	defer ts.Close()
	t.Log("httptest.Server listens on", ts.URL)

	e2e := &EndpointTestSuite{
		cfg:   cfg,
		admin: admin,
	}
	suite.Run(t, e2e)
}

type EndpointTestSuite struct {
	suite.Suite

	cfg   *IntegrationConfig
	admin *client.Client

	user   *model.User
	client *client.Client
}

func (self *EndpointTestSuite) SetupTest() {
	username := self.cfg.RandomUsername()
	user, err := self.admin.CreateUser(username, self.cfg.RegularPassword, false)
	self.Require().NoError(err)
	self.Require().NotNil(user)
	self.Require().Equal(username, user.Username, "Invalid username")
	self.user = user

	client := client.NewClient(self.cfg.BaseURL, user.Username,
		self.cfg.RegularPassword)
	self.Require().NotNil(client)
	self.client = client
}

func (self *EndpointTestSuite) TearDownTest() {
	if self.user == nil {
		return
	}

	self.Require().NoError(self.admin.DeleteUser(self.user.ID))
	self.user = nil
	self.client = nil
}

func (self *EndpointTestSuite) TestVersionEndpoint() {
	version, err := self.admin.Version()
	self.Require().NoError(err)
	self.Require().NotNil(version)

	self.NotEmpty(version.Version, "Version should not be empty")
	self.NotEmpty(version.Commit, "Commit should not be empty")
	self.NotEmpty(version.OS, "OS should not be empty")
}

func (self *EndpointTestSuite) TestInvalidCredentials() {
	api := client.NewClient(self.cfg.BaseURL, "invalid", "invalid")
	self.Require().NotNil(api)

	_, err := api.Users()
	self.Require().Error(err, "Using bad credentials should raise an error")
	self.ErrorIs(err, client.ErrNotAuthorized,
		`A "Not Authorized" error should be raised`)
}

func (self *EndpointTestSuite) TestGetMeEndpoint() {
	user, err := self.admin.Me()
	self.Require().NoError(err)
	self.Equal(self.cfg.AdminUsername, user.Username)
}

func (self *EndpointTestSuite) TestGetUsersEndpointAsAdmin() {
	users, err := self.admin.Users()
	self.Require().NoError(err)

	self.Require().NotEmpty(users, "Users should not be empty")
	self.NotEmpty(users[0].ID, "Invalid userID")
	self.Equal(self.cfg.AdminUsername, users[0].Username, "Invalid username")
	self.Empty(users[0].Password, "Invalid password")
	self.Equal("en_US", users[0].Language, "Invalid language")
	self.Equal("light_serif", users[0].Theme, "Invalid theme")
	self.Equal("UTC", users[0].Timezone, "Invalid timezone")
	self.True(users[0].IsAdmin, "Invalid role")
	self.Equal(100, users[0].EntriesPerPage, "Invalid entries per page")
	self.Equal("standalone", users[0].DisplayMode, "Invalid web app display mode")
	self.Equal("tap", users[0].GestureNav, "Invalid gesture navigation")
	self.Equal(265, users[0].DefaultReadingSpeed, "Invalid default reading speed")
	self.Equal(500, users[0].CJKReadingSpeed, "Invalid cjk reading speed")
}

func (self *EndpointTestSuite) TestGetUsersEndpointAsRegularUser() {
	_, err := self.client.Users()
	self.Require().Error(err,
		"Regular users should not have access to the users endpoint")
}

func (self *EndpointTestSuite) TestCreateUserEndpointAsAdmin() {
	self.Empty(self.user.Password, "Invalid password")
	self.Equal("en_US", self.user.Language, "Invalid language")
	self.Equal("light_serif", self.user.Theme, "Invalid theme")
	self.Equal("UTC", self.user.Timezone, "Invalid timezone")
	self.False(self.user.IsAdmin, "Invalid role")
	self.Equal(100, self.user.EntriesPerPage, "Invalid entries per page")
	self.Equal("standalone", self.user.DisplayMode,
		"Invalid web app display mode")
	self.Equal("tap", self.user.GestureNav, "Invalid gesture navigation")
	self.Equal(265, self.user.DefaultReadingSpeed,
		"Invalid default reading speed")
	self.Equal(500, self.user.CJKReadingSpeed, "Invalid cjk reading speed")
}

func (self *EndpointTestSuite) TestCreateUserEndpointAsRegularUser() {
	_, err := self.client.CreateUser(self.cfg.RandomUsername(),
		self.cfg.RegularPassword, false)
	self.Require().Error(err,
		"Regular users should not have access to the create user endpoint")
}

func (self *EndpointTestSuite) TestCannotCreateDuplicateUser() {
	_, err := self.admin.CreateUser(self.cfg.AdminUsername,
		self.cfg.AdminPassword, true)
	self.Require().Error(err, "Duplicated users should not be allowed")
}

func (self *EndpointTestSuite) TestRemoveUserEndpointAsAdmin() {
	user, err := self.admin.CreateUser(self.cfg.RandomUsername(),
		self.cfg.RegularPassword, false)
	self.Require().NoError(err)
	self.Require().NotNil(user)
	self.Require().NoError(self.admin.DeleteUser(user.ID))
}

func (self *EndpointTestSuite) TestRemoveUserEndpointAsRegularUser() {
	err := self.client.DeleteUser(self.user.ID)
	self.Require().Error(err,
		"Regular users should not have access to the remove user endpoint")
}

func (self *EndpointTestSuite) TestGetUserByIDEndpointAsAdmin() {
	user, err := self.admin.Me()
	self.Require().NoError(err)
	self.Require().NotNil(user)

	userByID, err := self.admin.UserByID(user.ID)
	self.Require().NoError(err)
	self.Require().NotNil(userByID)

	self.Equal(user.ID, userByID.ID, "Invalid userID")
	self.Equal(user.Username, userByID.Username, "Invalid username")
	self.Empty(userByID.Password, "The password field must be empty")
	self.Equal(user.Language, userByID.Language, "Invalid language")
	self.Equal(user.Theme, userByID.Theme, "Invalid theme")
	self.Equal(user.Timezone, userByID.Timezone, "Invalid timezone")
	self.Equal(user.IsAdmin, userByID.IsAdmin, "Invalid role")
	self.Equal(user.EntriesPerPage, userByID.EntriesPerPage,
		"Invalid entries per page")
	self.Equal(user.DisplayMode, userByID.DisplayMode,
		"Invalid web app display mode")
	self.Equal(user.GestureNav, userByID.GestureNav,
		"Invalid gesture navigation")
	self.Equal(user.DefaultReadingSpeed, userByID.DefaultReadingSpeed,
		"Invalid default reading speed")
	self.Equal(user.CJKReadingSpeed, userByID.CJKReadingSpeed,
		"Invalid cjk reading speed")
	self.Equal(user.EntryDirection, userByID.EntryDirection,
		"Invalid entry direction")
	self.Equal(user.EntryOrder, userByID.EntryOrder, "Invalid entry order")
}

func (self *EndpointTestSuite) TestGetUserByIDEndpointAsRegularUser() {
	_, err := self.client.UserByID(self.user.ID)
	self.Require().Error(err,
		"Regular users should not have access to the user by ID endpoint")
}

func (self *EndpointTestSuite) TestGetUserByUsernameEndpointAsAdmin() {
	user, err := self.admin.Me()
	self.Require().NoError(err)
	self.Require().NotNil(user)

	userByUsername, err := self.admin.UserByUsername(user.Username)
	self.Require().NoError(err)

	self.Equal(user.ID, userByUsername.ID, "Invalid userID")
	self.Equal(user.Username, userByUsername.Username, "Invalid username")
	self.Empty(userByUsername.Password, "The password field must be empty")
	self.Equal(user.Language, userByUsername.Language, "Invalid language")
	self.Equal(user.Theme, userByUsername.Theme, "Invalid theme")
	self.Equal(user.Timezone, userByUsername.Timezone, "Invalid timezone")
	self.Equal(user.IsAdmin, userByUsername.IsAdmin, "Invalid role")
	self.Equal(user.EntriesPerPage, userByUsername.EntriesPerPage,
		"Invalid entries per page")
	self.Equal(user.DisplayMode, userByUsername.DisplayMode, "Invalid web app display mode")
	self.Equal(user.GestureNav, userByUsername.GestureNav,
		"Invalid gesture navigation")
	self.Equal(user.DefaultReadingSpeed, userByUsername.DefaultReadingSpeed,
		"Invalid default reading speed")
	self.Equal(user.CJKReadingSpeed, userByUsername.CJKReadingSpeed,
		"Invalid cjk reading speed")
	self.Equal(user.EntryDirection, userByUsername.EntryDirection,
		"Invalid entry direction")
}

func (self *EndpointTestSuite) TestGetUserByUsernameEndpointAsRegularUser() {
	_, err := self.client.UserByUsername(self.user.Username)
	self.Require().Error(err,
		"Regular users should not have access to the user by username endpoint")
}

func (self *EndpointTestSuite) TestUpdateUserEndpoint() {
	const js = "alert('Hello, World!');"

	tests := []struct {
		name    string
		req     model.UserModificationRequest
		wantErr bool
		assert  func(updatedUser *model.User)
	}{
		{
			name: "mark_read_on_scroll",
			req: model.UserModificationRequest{
				MarkReadOnScroll: model.SetOptionalField(true),
			},
			assert: func(updatedUser *model.User) {
				self.True(updatedUser.Extra.MarkReadOnScroll)
			},
		},
		{
			name: "default theme",
			req: model.UserModificationRequest{
				Theme: model.SetOptionalField("dark_serif"),
			},
			assert: func(updatedUser *model.User) {
				self.Equal("dark_serif", updatedUser.Theme, "Invalid theme")
			},
		},
		{
			name: "external fonts",
			req: model.UserModificationRequest{
				ExternalFontHosts: model.SetOptionalField("  fonts.example.org  "),
			},
			assert: func(updatedUser *model.User) {
				self.Equal("fonts.example.org", updatedUser.ExternalFontHosts,
					"Invalid external font hosts")
			},
		},
		{
			name: "invalid external fonts",
			req: model.UserModificationRequest{
				ExternalFontHosts: model.SetOptionalField("'self' *"),
			},
			wantErr: true,
		},
		{
			name: "custom js",
			req: model.UserModificationRequest{
				CustomJS: model.SetOptionalField(js),
			},
			assert: func(updatedUser *model.User) {
				self.Equal(js, updatedUser.CustomJS, "Invalid custom JS")
			},
		},
		{
			name: "invalid theme",
			req: model.UserModificationRequest{
				Theme: model.SetOptionalField("invalid_theme"),
			},
			wantErr: true,
		},
		{
			name: "AlwaysOpenExternalLinks",
			req: model.UserModificationRequest{
				AlwaysOpenExternalLinks: model.SetOptionalField(true),
			},
			assert: func(updatedUser *model.User) {
				self.True(updatedUser.Extra.AlwaysOpenExternalLinks)
			},
		},
	}

	for _, tt := range tests {
		self.Run(tt.name, func() {
			updatedUser, err := self.client.UpdateUser(self.user.ID, &tt.req)
			if tt.wantErr {
				self.T().Log(err)
				self.Require().Error(err)
				return
			}
			tt.assert(updatedUser)
		})
	}
}

func (self *EndpointTestSuite) TestRegularUsersCannotUpdateOtherUsers() {
	adminUser, err := self.admin.Me()
	self.Require().NoError(err)
	self.Require().NotNil(adminUser)

	userUpdateRequest := model.UserModificationRequest{
		Theme: model.SetOptionalField("dark_serif"),
	}

	_, err = self.client.UpdateUser(adminUser.ID, &userUpdateRequest)
	self.Require().Error(err,
		"Regular users should not be able to update other users")
}

func (self *EndpointTestSuite) TestAPIKeysEndpoint() {
	apiKeys, err := self.client.APIKeys()
	self.Require().NoError(err)
	self.Empty(apiKeys, "Expected no API keys")

	// Create an API key for the user.
	apiKey, err := self.client.CreateAPIKey("Test API Key")
	self.Require().NoError(err)
	self.NotZero(apiKey.ID, "Invalid API key ID")
	self.Equal(self.user.ID, apiKey.UserID, "Invalid user ID for API key")
	self.NotEmpty(apiKey.Token, "Invalid API key token")
	self.Equal("Test API Key", apiKey.Description, "Invalid API key description")

	// Create a duplicate API key with the same description.
	_, err = self.client.CreateAPIKey("Test API Key")
	self.T().Log(err)
	self.Require().Error(err,
		"Creating a duplicate API key with the same description should raise an error")

	// Fetch the API keys again.
	apiKeys, err = self.client.APIKeys()
	self.Require().NoError(err)
	self.Equal([]*model.APIKey{apiKey}, apiKeys)

	// Create a new client using the API key.
	apiKeyClient := client.NewClient(self.cfg.BaseURL, apiKey.Token)
	self.Require().NotNil(apiKeyClient)

	// Fetch the user using the API key client.
	user, err := apiKeyClient.Me()
	self.Require().NoError(err)

	// Verify the user matches the regular test user.
	self.Equal(self.user.ID, user.ID)

	// Delete the API key.
	err = self.client.DeleteAPIKey(apiKey.ID)
	self.Require().NoError(err)

	// Verify the API key is deleted.
	apiKeys, err = self.client.APIKeys()
	self.Require().NoError(err)
	self.Empty(apiKeys, "Expected no API keys after deletion")

	// Try to delete the API key again, it should return an error.
	err = self.client.DeleteAPIKey(apiKey.ID)
	self.T().Log(err)
	self.Require().ErrorIs(err, client.ErrNotFound,
		"Deleting a non-existent API key should raise 'not found' error")

	// Try to create an API key with an empty description.
	_, err = self.client.CreateAPIKey("")
	self.T().Log(err)
	self.Require().Error(err,
		"Creating an API key with an empty description should raise an error")
}

func (self *EndpointTestSuite) TestMarkUserAsReadEndpoint() {
	feedID := self.createFeed()
	self.Require().NoError(self.client.MarkAllAsRead(self.user.ID))
	self.checkFeedIsRead(feedID)
}

func (self *EndpointTestSuite) createFeed() int64 {
	self.T().Helper()
	return self.createFeedWith(model.FeedCreationRequest{})
}

func (self *EndpointTestSuite) createFeedWith(r model.FeedCreationRequest,
) int64 {
	self.T().Helper()

	if r.FeedURL == "" {
		r.FeedURL = self.cfg.FeedURL
	}

	feedID, err := self.client.CreateFeed(&r)
	self.Require().NoError(err)
	self.NotZero(feedID, "Invalid feedID")
	return feedID
}

func (self *EndpointTestSuite) checkFeedIsRead(feedID int64) {
	self.T().Helper()

	results, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(results)
	self.T().Log("Got entries:", len(results.Entries))

	i := slices.IndexFunc(results.Entries, func(entry *model.Entry) bool {
		return entry.Status != model.EntryStatusRead
	})

	if !self.Equal(-1, i) {
		entry := results.Entries[i]
		self.T().Logf("Status for entry %d was %q instead of %q",
			entry.ID, entry.Status, model.EntryStatusRead)
	}
}

func (self *EndpointTestSuite) TestCannotMarkUserAsReadAsOtherUser() {
	adminUser, err := self.admin.Me()
	self.Require().NoError(err)
	self.Require().NotNil(adminUser)

	err = self.client.MarkAllAsRead(adminUser.ID)
	self.Require().Error(err,
		"Non-admin users should not be able to mark another user as read")
}

func (self *EndpointTestSuite) TestCreateCategoryEndpoint() {
	category := self.createCategory()
	self.NotEmpty(category.ID, "Invalid categoryID")
	self.Positive(category.UserID, "Invalid userID")
	self.Equal("My category", category.Title, "Invalid title")
	self.False(category.HideGlobally, "Invalid hide globally value")
}

func (self *EndpointTestSuite) createCategory() *model.Category {
	self.T().Helper()
	category, err := self.client.CreateCategory("My category")
	self.Require().NoError(err)
	self.Require().NotNil(category)
	return category
}

func (self *EndpointTestSuite) TestCreateCategoryWithEmptyTitle() {
	_, err := self.client.CreateCategory("")
	self.T().Log(err)
	self.Require().Error(err,
		"Creating a category with an empty title should raise an error")
}

func (self *EndpointTestSuite) TestCannotCreateDuplicatedCategory() {
	category := self.createCategory()
	_, err := self.client.CreateCategory(category.Title)
	self.T().Log(err)
	self.Require().Error(err, "Duplicated categories should not be allowed")
}

func (self *EndpointTestSuite) TestCreateCategoryWithOptions() {
	categoryCreate := model.CategoryCreationRequest{
		Title:        "My category",
		HideGlobally: true,
	}
	newCategory, err := self.client.CreateCategoryWithOptions(&categoryCreate)
	self.Require().NoError(err,
		"Creating a category with options should not raise an error")

	categories, err := self.client.Categories()
	self.Require().NoError(err)

	i := slices.IndexFunc(categories, func(c *model.Category) bool {
		return c.ID == newCategory.ID
	})
	self.Require().GreaterOrEqual(i, 0)

	category := categories[i]
	self.Equal(newCategory.Title, category.Title, "Invalid title")
	self.True(category.HideGlobally, "Invalid hide globally value")
}

func (self *EndpointTestSuite) TestUpdateCategoryEndpoint() {
	category := self.createCategory()

	const title = "new title"
	updatedCategory, err := self.client.UpdateCategory(category.ID, title)
	self.Require().NoError(err)
	self.Equal(category.ID, updatedCategory.ID, "Invalid categoryID")
	self.Equal(self.user.ID, updatedCategory.UserID, "Invalid userID")
	self.Equal(title, updatedCategory.Title, "Invalid title")
	self.False(updatedCategory.HideGlobally, "Invalid hide globally value")
}

func (self *EndpointTestSuite) TestUpdateCategoryWithOptions() {
	categoryCreate := model.CategoryCreationRequest{Title: "My category"}
	newCategory, err := self.client.CreateCategoryWithOptions(&categoryCreate)
	self.Require().NoError(err,
		"Creating a category with options should not raise an error")

	const title = "new title"
	categoryModify := model.CategoryModificationRequest{
		Title: model.SetOptionalField(title),
	}
	updatedCategory, err := self.client.UpdateCategoryWithOptions(
		newCategory.ID, &categoryModify)
	self.Require().NoError(err)
	self.Equal(newCategory.ID, updatedCategory.ID, "Invalid categoryID")
	self.Equal(title, updatedCategory.Title, "Invalid title")
	self.False(updatedCategory.HideGlobally, "Invalid hide globally value")

	categoryModify = model.CategoryModificationRequest{
		HideGlobally: model.SetOptionalField(true),
	}
	updatedCategory, err = self.client.UpdateCategoryWithOptions(
		newCategory.ID, &categoryModify)
	self.Require().NoError(err)
	self.Equal(newCategory.ID, updatedCategory.ID, "Invalid categoryID")
	self.Equal(title, updatedCategory.Title, "Invalid title")
	self.True(updatedCategory.HideGlobally, "Invalid hide globally value")

	categoryModify = model.CategoryModificationRequest{
		HideGlobally: model.SetOptionalField(false),
	}
	updatedCategory, err = self.client.UpdateCategoryWithOptions(
		newCategory.ID, &categoryModify)
	self.Require().NoError(err)
	self.Equal(newCategory.ID, updatedCategory.ID, "Invalid categoryID")
	self.Equal(title, updatedCategory.Title, "Invalid title")
	self.False(updatedCategory.HideGlobally, "Invalid hide globally value")
}

func (self *EndpointTestSuite) TestUpdateInexistingCategory() {
	_, err := self.admin.UpdateCategory(123456789, "new title")
	self.T().Log(err)
	self.Require().Error(err,
		"Updating an inexisting category should raise an error")
}

func (self *EndpointTestSuite) TestDeleteCategoryEndpoint() {
	category := self.createCategory()
	self.Require().NoError(self.client.DeleteCategory(category.ID))
}

func (self *EndpointTestSuite) TestCannotDeleteInexistingCategory() {
	err := self.admin.DeleteCategory(123456789)
	self.T().Log(err)
	self.Require().Error(err,
		"Deleting an inexisting category should raise an error")
}

func (self *EndpointTestSuite) TestCannotDeleteCategoryOfAnotherUser() {
	category := self.createCategory()
	err := self.admin.DeleteCategory(category.ID)
	self.T().Log(err)
	self.Require().Error(err,
		"Regular users should not be able to delete categories of other users")
}

func (self *EndpointTestSuite) TestGetCategoriesEndpoint() {
	category := self.createCategory()

	categories, err := self.client.Categories()
	self.Require().NoError(err)
	self.Len(categories, 2, "Invalid number of categories")
	self.Equal(self.user.ID, categories[0].UserID, "Invalid userID")
	self.Equal("All", categories[0].Title, "Invalid title")
	self.Equal(category.ID, categories[1].ID)
	self.Equal(self.user.ID, categories[1].UserID, "Invalid userID")
	self.Equal("My category", categories[1].Title, "Invalid title")
}

func (self *EndpointTestSuite) TestMarkCategoryAsReadEndpoint() {
	category := self.createCategory()
	feedID := self.createFeedWith(model.FeedCreationRequest{
		CategoryID: category.ID,
	})
	self.Require().NoError(self.client.MarkCategoryAsRead(category.ID))
	self.checkFeedIsRead(feedID)
}

func (self *EndpointTestSuite) TestCreateFeedEndpoint() {
	category := self.createCategory()
	self.createFeedWith(model.FeedCreationRequest{
		CategoryID: category.ID,
	})
}

func (self *EndpointTestSuite) TestCreateFeedEndpoint_2entries() {
	feedURL, err := url.Parse(self.cfg.FeedURL)
	self.Require().NoError(err)

	ref, err := url.Parse("/2entries.xml")
	self.Require().NoError(err)
	feedURL = feedURL.ResolveReference(ref)

	self.T().Log(feedURL)
	feedID := self.createFeedWith(
		model.FeedCreationRequest{FeedURL: feedURL.String()})

	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(result)
	self.Len(result.Entries, 2)
}

func (self *EndpointTestSuite) TestCreateFeedEndpoint_2hash() {
	feedURL, err := url.Parse(self.cfg.FeedURL)
	self.Require().NoError(err)

	ref, err := url.Parse("/2hash.xml")
	self.Require().NoError(err)
	feedURL = feedURL.ResolveReference(ref)

	self.T().Log(feedURL)
	feedID := self.createFeedWith(
		model.FeedCreationRequest{FeedURL: feedURL.String()})

	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(result)
	self.Len(result.Entries, 1)
}

func (self *EndpointTestSuite) TestCannotCreateDuplicatedFeed() {
	self.createFeed()
	_, err := self.client.CreateFeed(&model.FeedCreationRequest{
		FeedURL: self.cfg.FeedURL,
	})
	self.T().Log(err)
	self.Require().Error(err, "Duplicated feeds should not be allowed")
}

func (self *EndpointTestSuite) TestCreateFeedWithInexistingCategory() {
	_, err := self.client.CreateFeed(&model.FeedCreationRequest{
		FeedURL:    self.cfg.FeedURL,
		CategoryID: 123456789,
	})
	self.T().Log(err)
	self.Require().Error(err,
		"Creating a feed with an inexisting category should raise an error")
}

func (self *EndpointTestSuite) TestCreateFeedWithEmptyFeedURL() {
	_, err := self.admin.CreateFeed(&model.FeedCreationRequest{})
	self.T().Log(err)
	self.Require().Error(err,
		"Creating a feed with an empty feed URL should raise an error")
}

func (self *EndpointTestSuite) TestCreateFeedWithInvalidFeedURL() {
	_, err := self.client.CreateFeed(&model.FeedCreationRequest{
		FeedURL: "invalid_feed_url",
	})
	self.T().Log(err)
	self.Require().Error(err,
		"Creating a feed with an invalid feed URL should raise an error")
}

func (self *EndpointTestSuite) TestCreateDisabledFeed() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		Disabled: true,
	})

	feed, err := self.client.Feed(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(feed)
	self.True(feed.Disabled, "The feed should be disabled")
}

func (self *EndpointTestSuite) TestCreateFeedWithDisabledHTTPCache() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		IgnoreHTTPCache: true,
	})

	feed, err := self.client.Feed(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(feed)
	self.True(feed.IgnoreHTTPCache, "The feed should ignore the HTTP cache")
}

func (self *EndpointTestSuite) TestCreateFeedWithScraperRule() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		ScraperRules: "article",
	})

	feed, err := self.client.Feed(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(feed)
	self.Equal("article", feed.ScraperRules,
		`The feed should have the scraper rules set to "article"`)
}

func (self *EndpointTestSuite) TestUpdateFeedEndpoint() {
	const url = "https://example.org/feed.xml"
	feedID := self.createFeed()
	feedModify := model.FeedModificationRequest{
		FeedURL: model.SetOptionalField(url),
	}

	updatedFeed, err := self.client.UpdateFeed(feedID, &feedModify)
	self.Require().NoError(err)
	self.Require().NotNil(updatedFeed)
	self.Equal(url, updatedFeed.FeedURL, "Invalid feed URL")
}

func (self *EndpointTestSuite) TestCannotHaveDuplicateFeedWhenUpdatingFeed() {
	self.createFeed()

	feedURL, err := url.Parse(self.cfg.FeedURL)
	self.Require().NoError(err)
	ref, err := url.Parse("/2entries.xml")
	self.Require().NoError(err)
	feedURL = feedURL.ResolveReference(ref)

	self.T().Log(feedURL)
	feedID := self.createFeedWith(model.FeedCreationRequest{
		FeedURL: feedURL.String(),
	})

	feedModify := model.FeedModificationRequest{
		FeedURL: model.SetOptionalField(self.cfg.FeedURL),
	}

	_, err = self.client.UpdateFeed(feedID, &feedModify)
	self.T().Log(err)
	self.Require().Error(err, "Duplicated feeds should not be allowed")
}

func (self *EndpointTestSuite) TestUpdateFeedWithInvalidCategory() {
	feedID := self.createFeed()

	feedModify := model.FeedModificationRequest{
		CategoryID: model.SetOptionalField(int64(123456789)),
	}

	_, err := self.client.UpdateFeed(feedID, &feedModify)
	self.T().Log(err)
	self.Require().Error(err,
		"Updating a feed with an inexisting category should raise an error")
}

func (self *EndpointTestSuite) TestUpdateFeedEndpoint_CommentsURLTemplate() {
	tests := []struct {
		name        string
		commentsURL string
		wantErr     bool
	}{
		{
			name:        "absolute",
			commentsURL: "/comments/",
		},
		{
			name:        "templated",
			commentsURL: "{{ .URL }}/comments/",
		},
		{
			name:        "invalid",
			commentsURL: "{{ notexists }}/comments/",
			wantErr:     true,
		},
		{
			name:        "replace",
			commentsURL: `{{ replace .URL "/clanek/" "/diskuze/" }}`,
		},
	}

	feedID := self.createFeed()
	for _, tt := range tests {
		self.Run(tt.name, func() {
			r := model.FeedModificationRequest{
				CommentsURLTemplate: model.SetOptionalField(tt.commentsURL),
			}

			feed, err := self.client.UpdateFeed(feedID, &r)
			if tt.wantErr {
				self.T().Log(err)
				self.Require().Error(err)
				return
			}

			self.Require().NoError(err)
			self.Require().NotNil(feed)
			self.Equal(tt.commentsURL, feed.Extra.CommentsURLTemplate)
		})
	}
}

func (self *EndpointTestSuite) TestMarkFeedAsReadEndpoint() {
	feedID := self.createFeed()
	self.Require().NoError(self.client.MarkFeedAsRead(feedID))
	self.checkFeedIsRead(feedID)
}

func (self *EndpointTestSuite) TestFetchCountersEndpoint() {
	feedID := self.createFeed()

	counters, err := self.client.FetchCounters()
	self.Require().NoError(err)
	self.Require().NotNil(counters)

	self.Zero(counters.ReadCounters[feedID], "Invalid read counter")
	self.Positive(counters.UnreadCounters[feedID], "Invalid unread counter")
}

func (self *EndpointTestSuite) TestDeleteFeedEndpoint() {
	feedID := self.createFeed()
	err := self.client.DeleteFeed(feedID)
	self.Require().NoError(err)
}

func (self *EndpointTestSuite) TestRefreshAllFeedsEndpoint() {
	self.createFeed()
	self.Require().NoError(self.client.RefreshAllFeeds())
}

func (self *EndpointTestSuite) TestRefreshFeedEndpoint() {
	feedID := self.createFeed()
	self.Require().NoError(self.client.RefreshFeed(feedID))
}

func (self *EndpointTestSuite) TestRefreshFeedEndpoint_IgnoreHTTPCache() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		IgnoreHTTPCache: true,
	})
	self.Require().NoError(self.client.RefreshFeed(feedID))
}

func (self *EndpointTestSuite) TestRefreshFeedEndpoint_flushHistory() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		IgnoreHTTPCache: true,
	})
	self.Require().NoError(self.client.MarkFeedAsRead(feedID))
	self.Require().NoError(self.client.FlushHistory())
	self.Require().NoError(self.client.RefreshFeed(feedID))
}

func (self *EndpointTestSuite) TestRefreshFeedEndpoint_markedRead() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		IgnoreHTTPCache: true,
	})
	self.Require().NoError(self.client.MarkFeedAsRead(feedID))
	self.Require().NoError(self.client.RefreshFeed(feedID))
}

func (self *EndpointTestSuite) TestRefreshFeedEndpoint_dedup() {
	self.createFeed()

	feedURL, err := url.Parse(self.cfg.FeedURL)
	self.Require().NoError(err)
	ref, err := url.Parse("/empty.xml")
	self.Require().NoError(err)
	feedURL = feedURL.ResolveReference(ref)

	self.T().Log(feedURL.String())
	feedID := self.createFeedWith(model.FeedCreationRequest{
		FeedURL:         feedURL.String(),
		IgnoreHTTPCache: true,
	})

	ref, err = url.Parse("/dups.xml")
	self.Require().NoError(err)
	feedURL = feedURL.ResolveReference(ref)

	self.T().Log(feedURL.String())
	feedModify := model.FeedModificationRequest{
		FeedURL: model.SetOptionalField(feedURL.String()),
	}
	updatedFeed, err := self.client.UpdateFeed(feedID, &feedModify)
	self.Require().NoError(err)
	self.Require().NotNil(updatedFeed)
	self.Equal(feedURL.String(), updatedFeed.FeedURL)

	self.Require().NoError(self.client.RefreshFeed(feedID))

	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	entry := result.Entries[0]
	self.Require().NotNil(entry)
	self.Equal(model.EntryStatusRead, entry.Status)
}

func (self *EndpointTestSuite) TestGetFeedEndpoint() {
	feedID := self.createFeed()
	feed, err := self.client.Feed(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(feed)
	self.Equal(feedID, feed.ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, feed.FeedURL, "Invalid feed URL")
	self.Equal(self.cfg.WebsiteURL, feed.SiteURL, "Invalid site URL")
	self.Equal(self.cfg.FeedTitle, feed.Title, "Invalid title")
}

func (self *EndpointTestSuite) TestGetFeedIcon() {
	feedID := self.createFeed()
	icon, err := self.client.FeedIcon(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(icon)
	self.NotEmpty(icon.MimeType, "Invalid mime type")
	self.NotEmpty(icon.Data, "Invalid data")

	icon, err = self.client.Icon(icon.ID)
	self.Require().NoError(err)
	self.Require().NotNil(icon)
	self.NotEmpty(icon.MimeType, "Invalid mime type")
	self.NotEmpty(icon.Data, "Invalid data")
}

func (self *EndpointTestSuite) TestGetFeedIconWithInexistingFeedID() {
	_, err := self.admin.FeedIcon(123456789)
	self.Require().Error(err, "Fetching the icon of an inexisting feed should raise an error")
}

func (self *EndpointTestSuite) TestGetFeedsEndpoint() {
	feedID := self.createFeed()
	feeds, err := self.client.Feeds()
	self.Require().NoError(err)
	self.Len(feeds, 1, "Invalid number of feeds")
	self.Equal(feedID, feeds[0].ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, feeds[0].FeedURL, "Invalid feed URL")
}

func (self *EndpointTestSuite) TestGetCategoryFeedsEndpoint() {
	category := self.createCategory()
	feedID := self.createFeedWith(model.FeedCreationRequest{
		CategoryID: category.ID,
	})

	feeds, err := self.client.CategoryFeeds(category.ID)
	self.Require().NoError(err)
	self.Require().Len(feeds, 1, "Invalid number of feeds")
	self.Equal(feedID, feeds[0].ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, feeds[0].FeedURL, "Invalid feed URL")
}

func (self *EndpointTestSuite) TestExportEndpoint() {
	self.createFeed()
	exportedData, err := self.client.Export()
	self.Require().NoError(err)
	self.NotEmpty(exportedData, "Invalid exported data")
	self.True(strings.HasPrefix(string(exportedData), "<?xml"),
		"Invalid OPML export")
}

func (self *EndpointTestSuite) TestImportEndpoint() {
	data := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Test Category">
      <outline title="Test" text="Test" xmlUrl="` + self.cfg.FeedURL + `" htmlUrl="` + self.cfg.WebsiteURL + `"></outline>
    </outline>
  </body>
</opml>`

	bytesReader := bytes.NewReader([]byte(data))
	self.Require().NoError(self.client.Import(io.NopCloser(bytesReader)))
}

func (self *EndpointTestSuite) TestDiscoverSubscriptionsEndpoint() {
	subscriptions, err := self.admin.Discover(self.cfg.WebsiteURL)
	self.Require().NoError(err)
	self.Require().NotEmpty(subscriptions, "Invalid number of subscriptions")
	self.Equal(self.cfg.SubscriptionTitle, subscriptions[0].Title,
		"Invalid title")
	self.Equal(self.cfg.FeedURL, subscriptions[0].URL, "Invalid URL")
}

func (self *EndpointTestSuite) TestDiscoverSubscriptionsWithInvalidURL() {
	_, err := self.admin.Discover("invalid_url")
	self.T().Log(err)
	self.Require().Error(err,
		"Discovering subscriptions with an invalid URL should raise an error")
}

func (self *EndpointTestSuite) TestDiscoverSubscriptionsWithNoSubscription() {
	_, err := self.admin.Discover(self.cfg.BaseURL)
	self.T().Log(err)
	self.Require().ErrorIs(err, client.ErrNotFound,
		"Discovering subscriptions with no subscription should raise a 404 error")
}

func (self *EndpointTestSuite) TestGetAllFeedEntriesEndpoint() {
	feedID := self.createFeed()
	results, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(results)
	self.T().Log("Got entries:", len(results.Entries))
	self.Require().NotEmpty(results.Entries, "Invalid number of entries")
	self.NotZero(results.Total, "Invalid total")
	self.Equal(feedID, results.Entries[0].Feed.ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, results.Entries[0].Feed.FeedURL,
		"Invalid feed URL")
	self.NotEmpty(results.Entries[0].Title, "Invalid title")
}

func (self *EndpointTestSuite) TestGetAllCategoryEntriesEndpoint() {
	category := self.createCategory()
	feedID := self.createFeedWith(model.FeedCreationRequest{
		CategoryID: category.ID,
	})

	results, err := self.client.CategoryEntries(category.ID, nil)
	self.Require().NoError(err)
	self.Require().NotNil(results)
	self.T().Log("Got entries:", len(results.Entries))
	self.Require().NotEmpty(results.Entries, "Invalid number of entries")
	self.NotZero(results.Total, "Invalid total")
	self.Equal(feedID, results.Entries[0].Feed.ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, results.Entries[0].Feed.FeedURL,
		"Invalid feed URL")
	self.NotEmpty(results.Entries[0].Title, "Invalid title")
}

func (self *EndpointTestSuite) TestGetAllEntriesEndpointWithFilter() {
	feedID := self.createFeed()

	result, err := self.client.Entries(&client.Filter{FeedID: feedID})
	self.Require().NoError(err)
	self.Require().NotNil(result)
	self.T().Log("Got entries:", len(result.Entries))
	self.Require().NotEmpty(result.Entries, "Invalid number of entries")
	self.NotZero(result.Total, "Invalid total")
	self.Equal(feedID, result.Entries[0].Feed.ID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, result.Entries[0].Feed.FeedURL,
		"Invalid feed URL")
	self.NotEmpty(result.Entries[0].Title, "Invalid title")

	recent, err := self.client.Entries(&client.Filter{
		Order:     "published_at",
		Direction: "desc",
	})
	self.Require().NoError(err)
	self.Require().NotNil(recent)
	self.T().Log("Got entries:", len(recent.Entries))
	self.Require().NotEmpty(recent.Entries, "Invalid number of entries")
	self.NotZero(recent.Total, "Invalid total")
	self.NotEqual(result.Entries[0].Title, recent.Entries[0].Title,
		"Invalid order, got the same title")

	searched, err := self.client.Entries(&client.Filter{Search: "2.2.2"})
	self.Require().NoError(err)
	self.Require().NotNil(searched)
	self.Equal(1, searched.Total, "Invalid total")

	_, err = self.client.Entries(&client.Filter{Status: "invalid"})
	self.T().Log(err)
	self.Require().Error(err, "Using invalid status should raise an error")

	_, err = self.client.Entries(&client.Filter{Direction: "invalid"})
	self.T().Log(err)
	self.Require().Error(err, "Using invalid direction should raise an error")

	_, err = self.client.Entries(&client.Filter{Order: "invalid"})
	self.T().Log(err)
	self.Require().Error(err, "Using invalid order should raise an error")
}

func (self *EndpointTestSuite) TestGetGlobalEntriesEndpoint() {
	feedID := self.createFeedWith(model.FeedCreationRequest{
		HideGlobally: true,
	})

	feedIDEntry, err := self.client.Feed(feedID)
	self.Require().NoError(err)
	self.Require().NotNil(feedIDEntry)
	self.True(feedIDEntry.HideGlobally,
		"Expected feed to have globally_hidden set to true")

	/* Not filtering on GloballyVisible should return all entries */
	feedEntries, err := self.client.Entries(&client.Filter{FeedID: feedID})
	self.Require().NoError(err)
	self.Require().NotNil(feedEntries)
	self.NotEmpty(feedEntries.Entries,
		"Expected entries but response contained none")

	/* Feed is hidden globally, so this should be empty */
	globallyVisibleEntries, err := self.client.Entries(
		&client.Filter{GloballyVisible: true})
	self.Require().NoError(err)
	self.Require().NotNil(globallyVisibleEntries)
	self.Empty(globallyVisibleEntries.Entries, "Expected no entries")
}

func (self *EndpointTestSuite) TestUpdateEnclosureEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)

	var entry *model.Entry
	var enclosure *model.Enclosure
	at := 1
	for _, entry = range result.Entries {
		if len(entry.Enclosures()) > at {
			enclosure = &entry.Enclosures()[at]
			break
		}
	}
	self.Require().NotNil(entry, "missing enclosure in feed")

	enclosure.MediaProgression = 20
	update := model.EnclosureUpdateRequest{
		MediaProgression: enclosure.MediaProgression,
	}
	err = self.client.UpdateEnclosure(entry.ID, int64(at), &update)
	self.Require().NoError(err)

	entry2, err := self.client.Entry(entry.ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry2)
	self.Equal(entry.Enclosures(), entry2.Enclosures(),
		"Failed to update media_progression")

	err = self.client.UpdateEnclosure(entry.ID, int64(len(entry.Enclosures())),
		&update)
	self.T().Log(err)
	self.Require().Error(err)
}

func (self *EndpointTestSuite) TestGetEnclosureEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)

	var entry *model.Entry
	var expectedEnclosure *model.Enclosure
	for _, entry = range result.Entries {
		if len(entry.Enclosures()) != 0 {
			expectedEnclosure = &entry.Enclosures()[0]
			break
		}
	}
	self.Require().NotNil(entry, "missing enclosure in feed")

	entry2, err := self.client.Entry(entry.ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry2)
	self.Require().NotEmpty(entry2.Enclosures())
	self.Equal(expectedEnclosure, &entry2.Enclosures()[0])
}

func (self *EndpointTestSuite) TestGetEntryEndpoints() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	entry, err := self.client.FeedEntry(feedID, result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.Equal(result.Entries[0].ID, entry.ID, "Invalid entryID")
	self.Equal(feedID, entry.FeedID, "Invalid feedID")
	self.Equal(self.cfg.FeedURL, entry.Feed.FeedURL, "Invalid feed URL")

	entry, err = self.client.Entry(result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.Equal(result.Entries[0].ID, entry.ID, "Invalid entryID")

	entry, err = self.client.CategoryEntry(
		result.Entries[0].Feed.Category.ID, result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.Equal(result.Entries[0].ID, entry.ID, "Invalid entryID")
}

func (self *EndpointTestSuite) TestUpdateEntryStatusEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	err = self.client.UpdateEntries([]int64{result.Entries[0].ID},
		model.EntryStatusRead)
	self.Require().NoError(err)

	entry, err := self.client.Entry(result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.Equal(model.EntryStatusRead, entry.Status, "Invalid status")
}

func (self *EndpointTestSuite) TestUpdateEntryEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, nil)
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	entryUpdate := model.EntryUpdateRequest{
		Title:   model.SetOptionalField("New title"),
		Content: model.SetOptionalField("New content"),
	}

	updatedEntry, err := self.client.UpdateEntry(
		result.Entries[0].ID, &entryUpdate)
	self.Require().NoError(err)
	self.Require().NotNil(updatedEntry)
	self.Equal("New title", updatedEntry.Title, "Invalid title")
	self.Equal("New content", updatedEntry.Content, "Invalid content")

	entry, err := self.client.Entry(result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.Equal("New title", entry.Title, "Invalid title")
	self.Equal("New content", entry.Content, "Invalid content")
}

func (self *EndpointTestSuite) TestToggleBookmarkEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, &client.Filter{Limit: 1})
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	self.Require().NoError(self.client.ToggleBookmark(result.Entries[0].ID))

	entry, err := self.client.Entry(result.Entries[0].ID)
	self.Require().NoError(err)
	self.Require().NotNil(entry)
	self.True(entry.Starred, "The entry should be bookmarked")
}

func (self *EndpointTestSuite) TestSaveEntryEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, &client.Filter{Limit: 1})
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	self.Require().ErrorIs(
		self.client.SaveEntry(result.Entries[0].ID), client.ErrBadRequest,
		"Saving an entry should raise a bad request error because no integration is configured")
}

func (self *EndpointTestSuite) TestFetchIntegrationsStatusEndpoint() {
	hasIntegrations, err := self.client.IntegrationsStatus()
	self.Require().NoError(err, "Failed to fetch integrations status")
	self.False(hasIntegrations,
		"New user should not have integrations configured")
}

func (self *EndpointTestSuite) TestFetchContentEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, &client.Filter{Limit: 1})
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	content, err := self.client.FetchEntryOriginalContent(result.Entries[0].ID)
	self.Require().NoError(err)
	self.NotEmpty(content, "Invalid content")
}

func (self *EndpointTestSuite) TestFlushHistoryEndpoint() {
	feedID := self.createFeed()
	result, err := self.client.FeedEntries(feedID, &client.Filter{Limit: 3})
	self.Require().NoError(err, "Failed to get entries")
	self.Require().NotNil(result)
	self.Require().NotEmpty(result.Entries)

	err = self.client.UpdateEntries(
		[]int64{result.Entries[0].ID, result.Entries[1].ID},
		model.EntryStatusRead)
	self.Require().NoError(err)

	self.Require().NoError(self.client.FlushHistory())

	readEntries, err := self.client.Entries(
		&client.Filter{Status: model.EntryStatusRead})
	self.Require().NoError(err)
	self.Require().NotNil(readEntries)
	self.Zero(readEntries.Total, "Invalid total")

	removedEntries, err := self.client.Entries(
		&client.Filter{Status: model.EntryStatusRemoved})
	self.Require().NoError(err)
	self.Require().NotNil(removedEntries)
	self.Equal(2, removedEntries.Total, "Invalid total")
}
