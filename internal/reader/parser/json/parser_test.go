// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miniflux.app/v2/internal/reader/parser"
)

func TestParseJsonFeedVersion1(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"icon": "https://micro.blog/jsonfeed/avatar.jpg",
		"favicon": "https://micro.blog/jsonfeed/favicon.png",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "https://example.org/second-item"
			},
			{
				"id": "1",
				"content_html": "<p>Hello, world!</p>",
				"url": "https://example.org/initial-post"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	assert.Equal(t, "My Example Feed", feed.Title)
	assert.Empty(t, feed.Description)
	assert.Equal(t, "https://example.org/feed.json", feed.FeedURL)
	assert.Equal(t, "https://example.org/", feed.SiteURL)
	assert.Equal(t, "https://micro.blog/jsonfeed/favicon.png", feed.IconURL)

	require.Len(t, feed.Entries, 2)
	assert.Equal(t, "fc5dafea8dc10c48", feed.Entries[0].Hash)
	assert.Equal(t, "https://example.org/second-item", feed.Entries[0].URL)
	assert.Equal(t, "This is a second item.", feed.Entries[0].Content)

	assert.Equal(t, "cd96027c85652110", feed.Entries[1].Hash)
	assert.Equal(t, "https://example.org/initial-post", feed.Entries[1].URL)
	assert.Equal(t, "<p>Hello, world!</p>", feed.Entries[1].Content)
}

func TestParseFeedWithDescription(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"description": "This is a sample feed description.",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": []
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Description != "This is a sample feed description." {
		t.Errorf("Incorrect description, got: %s", feed.Description)
	}
}

func TestParsePodcast(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"user_comment": "This is a podcast feed. You can add this feed to your podcast client using the following URL: http://therecord.co/feed.json",
		"title": "The Record",
		"home_page_url": "http://therecord.co/",
		"feed_url": "http://therecord.co/feed.json",
		"items": [
			{
				"id": "http://therecord.co/chris-parrish",
				"title": "Special #1 - Chris Parrish",
				"url": "http://therecord.co/chris-parrish",
				"content_text": "Chris has worked at Adobe and as a founder of Rogue Sheep, which won an Apple Design Award for Postage. Chris’s new company is Aged & Distilled with Guy English — which shipped Napkin, a Mac app for visual collaboration. Chris is also the co-host of The Record. He lives on Bainbridge Island, a quick ferry ride from Seattle.",
				"content_html": "Chris has worked at <a href=\"http://adobe.com/\">Adobe</a> and as a founder of Rogue Sheep, which won an Apple Design Award for Postage. Chris’s new company is Aged & Distilled with Guy English — which shipped <a href=\"http://aged-and-distilled.com/napkin/\">Napkin</a>, a Mac app for visual collaboration. Chris is also the co-host of The Record. He lives on <a href=\"http://www.ci.bainbridge-isl.wa.us/\">Bainbridge Island</a>, a quick ferry ride from Seattle.",
				"summary": "Brent interviews Chris Parrish, co-host of The Record and one-half of Aged & Distilled.",
				"date_published": "2014-05-09T14:04:00-07:00",
				"attachments": [
					{
						"url": "http://therecord.co/downloads/The-Record-sp1e1-ChrisParrish.m4a",
						"mime_type": "audio/x-m4a",
						"size_in_bytes": 89970236,
						"duration_in_seconds": 6629
					}
				]
			}
		]
	}`

	feed, err := parser.ParseBytes("http://therecord.co/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "The Record" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}

	if feed.FeedURL != "http://therecord.co/feed.json" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "http://therecord.co/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Hash != "5d0218053a4082a8" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[0].Hash)
	}

	if feed.Entries[0].URL != "http://therecord.co/chris-parrish" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}

	if feed.Entries[0].Title != "Special #1 - Chris Parrish" {
		t.Errorf(`Incorrect entry title, got: "%s"`, feed.Entries[0].Title)
	}

	if feed.Entries[0].Content != `Chris has worked at <a href="http://adobe.com/">Adobe</a> and as a founder of Rogue Sheep, which won an Apple Design Award for Postage. Chris’s new company is Aged & Distilled with Guy English — which shipped <a href="http://aged-and-distilled.com/napkin/">Napkin</a>, a Mac app for visual collaboration. Chris is also the co-host of The Record. He lives on <a href="http://www.ci.bainbridge-isl.wa.us/">Bainbridge Island</a>, a quick ferry ride from Seattle.` {
		t.Errorf(`Incorrect entry content, got: "%s"`, feed.Entries[0].Content)
	}

	location, _ := time.LoadLocation("America/Vancouver")
	if !feed.Entries[0].Date.Equal(time.Date(2014, time.May, 9, 14, 4, 0, 0, location)) {
		t.Errorf("Incorrect entry date, got: %v", feed.Entries[0].Date)
	}

	if len(feed.Entries[0].Enclosures()) != 1 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}

	if feed.Entries[0].Enclosures()[0].URL != "http://therecord.co/downloads/The-Record-sp1e1-ChrisParrish.m4a" {
		t.Errorf("Incorrect enclosure URL, got: %s", feed.Entries[0].Enclosures()[0].URL)
	}

	if feed.Entries[0].Enclosures()[0].MimeType != "audio/x-m4a" {
		t.Errorf("Incorrect enclosure type, got: %s", feed.Entries[0].Enclosures()[0].MimeType)
	}

	if feed.Entries[0].Enclosures()[0].Size != 89970236 {
		t.Errorf("Incorrect enclosure length, got: %d", feed.Entries[0].Enclosures()[0].Size)
	}
}

func TestParseFeedWithFeedURLWithTrailingSpace(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json ",
		"items": []
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/feed.json" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}
}

func TestParseFeedWithRelativeFeedURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "/feed.json",
		"items": []
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/feed.json" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}
}

func TestParseFeedSiteURLWithTrailingSpace(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/ ",
		"feed_url": "https://example.org/feed.json",
		"items": []
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}
}

func TestParseFeedWithRelativeSiteURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "/home ",
		"feed_url": "https://example.org/feed.json",
		"items": []
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/home" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}
}

func TestParseFeedWithoutTitle(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "https://example.org/" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseFeedWithoutHomePage(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"feed_url": "https://example.org/feed.json",
		"title": "Some test",
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/feed.json" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseFeedWithoutFeedURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "Some test",
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/feed.json" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseItemWithoutAttachmentURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"user_comment": "This is a podcast feed. You can add this feed to your podcast client using the following URL: http://therecord.co/feed.json",
		"title": "The Record",
		"home_page_url": "http://therecord.co/",
		"feed_url": "http://therecord.co/feed.json",
		"items": [
			{
				"id": "http://therecord.co/chris-parrish",
				"title": "Special #1 - Chris Parrish",
				"url": "http://therecord.co/chris-parrish",
				"content_text": "Chris has worked at Adobe and as a founder of Rogue Sheep, which won an Apple Design Award for Postage. Chris’s new company is Aged & Distilled with Guy English — which shipped Napkin, a Mac app for visual collaboration. Chris is also the co-host of The Record. He lives on Bainbridge Island, a quick ferry ride from Seattle.",
				"date_published": "2014-05-09T14:04:00-07:00",
				"attachments": [
					{
						"url": "",
						"mime_type": "audio/x-m4a",
						"size_in_bytes": 0
					}
				]
			}
		]
	}`

	feed, err := parser.ParseBytes("http://therecord.co/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Enclosures()) != 0 {
		t.Errorf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}
}

func TestParseItemWithRelativeURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "Example",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2347259",
				"url": "something.html",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].URL != "https://example.org/something.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseItemWithExternalURLAndNoURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "Example",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "1234259",
				"external_url": "some_page.html"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].URL != "https://example.org/some_page.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseItemWithExternalURLAndURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "Example",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "1234259",
				"url": "https://example.org/article",
				"external_url": "https://example.org/another-article"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].URL != "https://example.org/article" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseItemWithLegacyAuthorField(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"user_comment": "This is a microblog feed. You can add this to your feed reader using the following URL: https://example.org/feed.json",
		"title": "Brent Simmons’s Microblog",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"author": {
			"name": "Brent Simmons",
			"url": "http://example.org/",
			"avatar": "https://example.org/avatar.png"
		},
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Author != "Brent Simmons" {
		t.Errorf("Incorrect entry author, got: %s", feed.Entries[0].Author)
	}
}

func TestParseItemWithMultipleAuthorFields(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1.1",
		"user_comment": "This is a microblog feed. You can add this to your feed reader using the following URL: https://example.org/feed.json",
		"title": "Brent Simmons’s Microblog",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"author": {
			"name": "Deprecated Author Field",
			"url": "http://example.org/",
			"avatar": "https://example.org/avatar.png"
		},
		"authors": [
			{
				"name": "Brent Simmons",
				"url": "http://example.org/",
				"avatar": "https://example.org/avatar.png"
			}
		],
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Author != "Brent Simmons, Deprecated Author Field" {
		t.Errorf("Incorrect entry author, got: %s", feed.Entries[0].Author)
	}
}

func TestParseItemWithMultipleDuplicateAuthors(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1.1",
		"title": "Example",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00",
				"authors": [
					{
						"name": "Author B",
						"url": "http://example.org/",
						"avatar": "https://example.org/avatar.png"
					},
					{
						"name": "Author A",
						"url": "http://example.org/",
						"avatar": "https://example.org/avatar.png"
					},
					{
						"name": "Author B",
						"url": "http://example.org/",
						"avatar": "https://example.org/avatar.png"
					}
				]
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Author != "Author A, Author B" {
		t.Errorf("Incorrect entry author, got: %s", feed.Entries[0].Author)
	}
}

func TestParseItemWithInvalidDate(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "Tomorrow"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	duration := time.Since(feed.Entries[0].Date)
	if duration.Seconds() > 1 {
		t.Errorf("Incorrect entry date, got: %v", feed.Entries[0].Date)
	}
}

func TestParseItemWithoutTitleButWithURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"url": "https://example.org/item"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "https://example.org/item" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseItemWithTooLongUnicodeTitle(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"title": "I’m riding my electric bike and came across this castle. It’s called “Schloss Richmond”. 🚴‍♂️"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Title) != 110 {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}

	if len([]rune(feed.Entries[0].Title)) != 93 {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseItemTitleWithXMLTags(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"title": "</example>"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "</example>" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseItemWithoutID(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"content_text": "Some text."
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Hash != "67395394917896a4" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[0].Hash)
	}
}

func TestParseItemTags(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"user_comment": "This is a microblog feed. You can add this to your feed reader using the following URL: https://example.org/feed.json",
		"title": "Brent Simmons’s Microblog",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"author": {
			"name": "Brent Simmons",
			"url": "http://example.org/",
			"avatar": "https://example.org/avatar.png"
		},
		"items": [
			{
				"id": "2347259",
				"url": "https://example.org/2347259",
				"content_text": "Cats are neat. \n\nhttps://example.org/cats",
				"date_published": "2016-02-09T14:22:00-07:00",
				"tags": [
					" tag 1",
					" ",
					"tag 2",
					"tag 2",
					"aaa"
				]
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, []string{"aaa", "tag 1", "tag 2"}, feed.Entries[0].Tags)
}

func TestParseFeedFavicon(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"favicon": "https://example.org/jsonfeed/favicon.png",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "https://example.org/second-item"
			},
			{
				"id": "1",
				"content_html": "<p>Hello, world!</p>",
				"url": "https://example.org/initial-post"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if feed.IconURL != "https://example.org/jsonfeed/favicon.png" {
		t.Errorf("Incorrect icon URL, got: %s", feed.IconURL)
	}
}

func TestParseFeedIcon(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"icon": "https://example.org/jsonfeed/icon.png",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "https://example.org/second-item"
			},
			{
				"id": "1",
				"content_html": "<p>Hello, world!</p>",
				"url": "https://example.org/initial-post"
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if feed.IconURL != "https://example.org/jsonfeed/icon.png" {
		t.Errorf("Incorrect icon URL, got: %s", feed.IconURL)
	}
}

func TestParseFeedWithRelativeAttachmentURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "https://example.org/",
		"feed_url": "https://example.org/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "https://example.org/second-item",
				"attachments": [
					{
						"url": "   /attachment.mp3  ",
						"mime_type": "audio/mpeg",
						"size_in_bytes": 123456
					}
				]
			}
		]
	}`

	feed, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries[0].Enclosures()) != 1 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}

	if feed.Entries[0].Enclosures()[0].URL != "https://example.org/attachment.mp3" {
		t.Errorf("Incorrect enclosure URL, got: %q", feed.Entries[0].Enclosures()[0].URL)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	data := `garbage`
	_, err := parser.ParseBytes("https://example.org/feed.json", []byte(data))
	if err == nil {
		t.Error("Parse should returns an error")
	}
}
