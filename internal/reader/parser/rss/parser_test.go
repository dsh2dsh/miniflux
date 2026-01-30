// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rss_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/parser"
)

func TestParseRss2Sample(t *testing.T) {
	data := `
		<?xml version="1.0"?>
		<rss version="2.0">
		<channel>
			<title>Liftoff News</title>
			<link>http://liftoff.msfc.nasa.gov/</link>
			<description>Liftoff to Space Exploration.</description>
			<image>
				<url>http://liftoff.msfc.nasa.gov/HomePageXtra/MeatBall.gif</url>
				<title>NASA</title>
				<link>http://liftoff.msfc.nasa.gov/</link>
			</image>
			<language>en-us</language>
			<pubDate>Tue, 10 Jun 2003 04:00:00 GMT</pubDate>
			<lastBuildDate>Tue, 10 Jun 2003 09:41:01 GMT</lastBuildDate>
			<docs>http://blogs.law.harvard.edu/tech/rss</docs>
			<generator>Weblog Editor 2.0</generator>
			<managingEditor>editor@example.com</managingEditor>
			<webMaster>webmaster@example.com</webMaster>
			<item>
				<title>Star City</title>
				<link>http://liftoff.msfc.nasa.gov/news/2003/news-starcity.asp</link>
				<description>How do Americans get ready to work with Russians aboard the International Space Station? They take a crash course in culture, language and protocol at Russia's &lt;a href="http://howe.iki.rssi.ru/GCTC/gctc_e.htm"&gt;Star City&lt;/a&gt;.</description>
				<pubDate>Tue, 03 Jun 2003 09:39:21 GMT</pubDate>
				<guid>http://liftoff.msfc.nasa.gov/2003/06/03.html#item573</guid>
			</item>
			<item>
				<description>Sky watchers in Europe, Asia, and parts of Alaska and Canada will experience a &lt;a href="http://science.nasa.gov/headlines/y2003/30may_solareclipse.htm"&gt;partial eclipse of the Sun&lt;/a&gt; on Saturday, May 31st.</description>
				<pubDate>Fri, 30 May 2003 11:06:42 GMT</pubDate>
				<guid>http://liftoff.msfc.nasa.gov/2003/05/30.html#item572</guid>
			</item>
			<item>
				<title>The Engine That Does More</title>
				<link>http://liftoff.msfc.nasa.gov/news/2003/news-VASIMR.asp</link>
				<description>Before man travels to Mars, NASA hopes to design new engines that will let us fly through the Solar System more quickly.  The proposed VASIMR engine would do that.</description>
				<pubDate>Tue, 27 May 2003 08:37:32 GMT</pubDate>
				<guid>http://liftoff.msfc.nasa.gov/2003/05/27.html#item571</guid>
			</item>
			<item>
				<title>Astronauts' Dirty Laundry</title>
				<link>http://liftoff.msfc.nasa.gov/news/2003/news-laundry.asp</link>
				<description>Compared to earlier spacecraft, the International Space Station has many luxuries, but laundry facilities are not one of them.  Instead, astronauts have other options.</description>
				<pubDate>Tue, 20 May 2003 08:56:02 GMT</pubDate>
				<guid>http://liftoff.msfc.nasa.gov/2003/05/20.html#item570</guid>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("http://liftoff.msfc.nasa.gov/rss.xml",
		[]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "Liftoff News" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}

	if feed.Description != "Liftoff to Space Exploration." {
		t.Errorf("Incorrect description, got: %s", feed.Description)
	}

	if feed.FeedURL != "http://liftoff.msfc.nasa.gov/rss.xml" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "http://liftoff.msfc.nasa.gov/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if feed.IconURL != "http://liftoff.msfc.nasa.gov/HomePageXtra/MeatBall.gif" {
		t.Errorf("Incorrect image URL, got: %s", feed.IconURL)
	}

	if len(feed.Entries) != 4 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	expectedDate := time.Date(2003, time.June, 3, 9, 39, 21, 0, time.UTC)
	if !feed.Entries[0].Date.Equal(expectedDate) {
		t.Errorf("Incorrect entry date, got: %v, want: %v", feed.Entries[0].Date, expectedDate)
	}

	if feed.Entries[0].Hash != "794444729ba5dcd0" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[0].Hash)
	}

	if feed.Entries[0].URL != "http://liftoff.msfc.nasa.gov/news/2003/news-starcity.asp" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}

	if feed.Entries[0].Title != "Star City" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}

	if feed.Entries[0].Content != `How do Americans get ready to work with Russians aboard the International Space Station? They take a crash course in culture, language and protocol at Russia's <a href="http://howe.iki.rssi.ru/GCTC/gctc_e.htm">Star City</a>.` {
		t.Errorf("Incorrect entry content, got: %s", feed.Entries[0].Content)
	}

	if feed.Entries[1].URL != "http://liftoff.msfc.nasa.gov/2003/05/30.html#item572" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[1].URL)
	}
}

func TestParseFeedWithFeedURLWithTrailingSpace(t *testing.T) {
	data := `<?xml version="1.0"?>
		<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<atom:link href="https://example.org/rss " type="application/rss+xml" rel="self"></atom:link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/ ", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/rss" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}
}

func TestParseFeedWithRelativeFeedURL(t *testing.T) {
	data := `<?xml version="1.0"?>
		<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<atom:link href="/rss" type="application/rss+xml" rel="self"></atom:link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/rss" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}
}

func TestParseFeedSiteURLWithTrailingSpace(t *testing.T) {
	data := `<?xml version="1.0"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/ </link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}
}

func TestParseFeedWithRelativeSiteURL(t *testing.T) {
	data := `<?xml version="1.0"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>/example </link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/example" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}
}

func TestParseFeedWithoutTitle(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "https://example.org/" {
		t.Errorf("Incorrect feed title, got: %s", feed.Title)
	}
}

func TestParseEntryWithoutTitleAndDescription(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Description != "" {
		t.Errorf("Expected empty feed description, got: %s", feed.Description)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Expected 1 entry, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Title != "https://example.org/item" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseEntryWithoutTitleButWithDescription(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<link>https://example.org/item</link>
				<description>
					This is the description
				</description>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Empty(t, feed.Entries[0].Title)
}

func TestParseEntryWithMediaTitle(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Entry Title</title>
				<link>https://example.org/item</link>
				<media:title>Media Title</media:title>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "Entry Title" {
		t.Errorf("Incorrect entry title, got: %q", feed.Entries[0].Title)
	}
}

func TestParseEntryWithDCTitleOnly(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/" xmlns:dc="http://purl.org/dc/elements/1.1/">
		<channel>
			<link>https://example.org/</link>
			<item>
				<dc:title>Entry Title</dc:title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "Entry Title" {
		t.Errorf("Incorrect entry title, got: %q", feed.Entries[0].Title)
	}
}

func TestParseFeedTitleWithHTMLEntity(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<title>Example &nbsp; Feed</title>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)
	assert.Equal(t, "Example \u00a0 Feed", feed.Title)
}

func TestParseFeedTitleWithUnicodeEntityAndCdata(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<title><![CDATA[Jenny’s Newsletter]]></title>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != `Jenny’s Newsletter` {
		t.Errorf(`Incorrect title, got: %q`, feed.Title)
	}
}

func TestParseItemTitleWithHTMLEntity(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<title>Example</title>
			<item>
				<title>&lt;/example&gt;</title>
				<link>http://www.example.org/entries/1</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "</example>" {
		t.Errorf(`Incorrect title, got: %q`, feed.Entries[0].Title)
	}
}

func TestParseItemTitleWithNumericCharacterReference(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<title>Example</title>
			<item>
				<title>&#931; &#xDF;</title>
				<link>http://www.example.org/article.html</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "Σ ß" {
		t.Errorf(`Incorrect title, got: %q`, feed.Entries[0].Title)
	}
}

func TestParseItemTitleWithDoubleEncodedEntities(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<title>Example</title>
			<item>
				<title>&amp;#39;Text&amp;#39;</title>
				<link>http://www.example.org/article.html</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, "&#39;Text&#39;", feed.Entries[0].Title)
}

func TestParseItemTitleWithWhitespaces(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<rss version="2.0">
	<channel>
		<title>Example</title>
		<link>http://example.org</link>
		<item>
			<title>
				Some Title
			</title>
			<link>http://www.example.org/entries/1</link>
		</item>
	</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "Some Title" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseItemTitleWithCDATA(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<rss version="2.0">
	<channel>
		<title>Example</title>
		<link>http://example.org</link>
		<item>
			<title><![CDATA[This is a title]]></title>
			<link>http://www.example.org/entries/1</link>
		</item>
	</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "This is a title" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseItemTitleWithInnerHTML(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<rss version="2.0">
	<channel>
		<title>Example</title>
		<link>http://example.org</link>
		<item>
			<title>Test: <b>bold</b></title>
			<link>http://www.example.org/entries/1</link>
		</item>
	</channel>
	</rss>`

	_, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.Error(t, err)
}

func TestParseEntryWithoutLink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<guid isPermalink="false">1234</guid>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, "https://example.org/", feed.Entries[0].URL)
	assert.Equal(t, "d8316e61d84f6ba4", feed.Entries[0].Hash)
}

func TestParseEntryWithoutLinkAndWithoutGUID(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Item 1</title>
			</item>
			<item>
				<title>Item 2</title>
				<pubDate>Wed, 02 Oct 2002 08:00:00 GMT</pubDate>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 2 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Hash != "9368225b3177b0d5" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[0].Hash)
	}

	if feed.Entries[1].Hash != "da56d7378b5b3596" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[1].Hash)
	}
}

func TestParseEntryWithOnlyGuidPermalink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<guid isPermaLink="true">https://example.org/some-article.html</guid>
			</item>
			<item>
				<guid>https://example.org/another-article.html</guid>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].URL != "https://example.org/some-article.html" {
		t.Errorf("Incorrect entry link, got: %s", feed.Entries[0].URL)
	}

	if feed.Entries[1].URL != "https://example.org/another-article.html" {
		t.Errorf("Incorrect entry link, got: %s", feed.Entries[1].URL)
	}
}

func TestParseEntryWithAtomLink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Test</title>
				<atom:link href="https://example.org/item" />
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].URL != "https://example.org/item" {
		t.Errorf("Incorrect entry link, got: %s", feed.Entries[0].URL)
	}
}

func TestParseEntryWithMultipleAtomLinks(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Test</title>
				<atom:link rel="payment" href="https://example.org/a" />
				<atom:link rel="alternate" href="https://example.org/b" />
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].URL != "https://example.org/b" {
		t.Errorf("Incorrect entry link, got: %s", feed.Entries[0].URL)
	}
}

func TestParseEntryWithoutLinkAndWithEnclosureURLs(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/feed</link>
			<item>
				<guid isPermalink="false">guid</guid>
				<enclosure url="https://audio-file" length="155844084" type="audio/mpeg" />
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, "https://audio-file", feed.Entries[0].URL)
}

func TestParseFeedURLWithAtomLink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<atom:link href="https://example.org/rss" type="application/rss+xml" rel="self"></atom:link>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/rss" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "https://example.org/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}
}

func TestParseFeedWithWebmaster(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<webMaster>webmaster@example.com</webMaster>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "webmaster@example.com"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseFeedWithManagingEditor(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<webMaster>webmaster@example.com</webMaster>
			<managingEditor>editor@example.com</managingEditor>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "editor@example.com"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseEntryWithAuthorAndInnerHTML(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<atom:link href="https://example.org/rss" type="application/rss+xml" rel="self"></atom:link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<author>by <a itemprop="url" class="author" rel="author" href="/author/foobar">Foo Bar</a></author>
			</item>
		</channel>
		</rss>`

	_, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.Error(t, err)
	t.Log(err)
}

func TestParseEntryWithAuthorAndCDATA(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<atom:link href="https://example.org/rss" type="application/rss+xml" rel="self"></atom:link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<author>
					<![CDATA[by Foo Bar]]>
				</author>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	expected := "by Foo Bar"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseEntryWithDublinCoreAuthor(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<dc:creator>Me (me@example.com)</dc:creator>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, "Me", feed.Entries[0].Author)
}

func TestParseEntryWithItunesAuthor(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<itunes:author>Someone</itunes:author>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "Someone"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseFeedWithItunesAuthor(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<itunes:author>Someone</itunes:author>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "Someone"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseFeedWithItunesOwner(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<itunes:owner>
				<itunes:name>John Doe</itunes:name>
				<itunes:email>john.doe@example.com</itunes:email>
			</itunes:owner>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "John Doe"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseFeedWithItunesOwnerEmail(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<itunes:owner>
				<itunes:email>john.doe@example.com</itunes:email>
			</itunes:owner>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := "john.doe@example.com"
	result := feed.Entries[0].Author
	if result != expected {
		t.Errorf("Incorrect entry author, got %q instead of %q", result, expected)
	}
}

func TestParseEntryWithDublinCoreDate(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
				<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
				<channel>
					<title>Example</title>
					<link>http://example.org/</link>
					<item>
						<title>Item 1</title>
						<link>http://example.org/item1</link>
						<description>Description.</description>
						<guid isPermaLink="false">UUID</guid>
						<dc:date>2002-09-29T23:40:06-05:00</dc:date>
					</item>
				</channel>
			</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	location, _ := time.LoadLocation("EST")
	expectedDate := time.Date(2002, time.September, 29, 23, 40, 0o6, 0, location)
	if !feed.Entries[0].Date.Equal(expectedDate) {
		t.Errorf("Incorrect entry date, got: %v, want: %v", feed.Entries[0].Date, expectedDate)
	}
}

func TestParseEntryWithContentEncoded(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		<channel>
			<title>Example</title>
			<link>http://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>http://example.org/item1</link>
				<description>Description.</description>
				<guid isPermaLink="false">UUID</guid>
				<content:encoded><![CDATA[<p><a href="http://www.example.org/">Example</a>.</p>]]></content:encoded>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Content != `<p><a href="http://www.example.org/">Example</a>.</p>` {
		t.Errorf("Incorrect entry content, got: %s", feed.Entries[0].Content)
	}
}

// https://www.rssboard.org/rss-encoding-examples
func TestParseEntryDescriptionWithEncodedHTMLTags(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		<channel>
			<title>Example</title>
			<link>http://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>http://example.org/item1</link>
				<description>this is &lt;b&gt;bold&lt;/b&gt;</description>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Content != `this is <b>bold</b>` {
		t.Errorf("Incorrect entry content, got: %q", feed.Entries[0].Content)
	}
}

// https://www.rssboard.org/rss-encoding-examples
func TestParseEntryWithDescriptionWithHTMLCDATA(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		<channel>
			<title>Example</title>
			<link>http://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>http://example.org/item1</link>
				<description><![CDATA[this is <b>bold</b>]]></description>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Content != `this is <b>bold</b>` {
		t.Errorf("Incorrect entry content, got: %q", feed.Entries[0].Content)
	}
}

// https://www.rssboard.org/rss-encoding-examples
func TestParseEntryDescriptionWithEncodingAngleBracketsInText(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		<channel>
			<title>Example</title>
			<link>http://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>http://example.org/item1</link>
				<description>5 &amp;lt; 8, ticker symbol &amp;lt;BIGCO&amp;gt;</description>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Content != `5 &lt; 8, ticker symbol &lt;BIGCO&gt;` {
		t.Errorf("Incorrect entry content, got: %q", feed.Entries[0].Content)
	}
}

// https://www.rssboard.org/rss-encoding-examples
func TestParseEntryDescriptionWithEncodingAngleBracketsWithinCDATASection(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		<channel>
			<title>Example</title>
			<link>http://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>http://example.org/item1</link>
				<description><![CDATA[5 &lt; 8, ticker symbol &lt;BIGCO&gt;]]></description>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Content != `5 &lt; 8, ticker symbol &lt;BIGCO&gt;` {
		t.Errorf("Incorrect entry content, got: %q", feed.Entries[0].Content)
	}
}

func TestParseEntryWithEnclosures(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
		<title>My Podcast Feed</title>
		<link>http://example.org</link>
		<author>some.email@example.org</author>
		<item>
			<title>Podcasting with RSS</title>
			<link>http://www.example.org/entries/1</link>
			<description>An overview of RSS podcasting</description>
			<pubDate>Fri, 15 Jul 2005 00:00:00 -0500</pubDate>
			<guid isPermaLink="true">http://www.example.org/entries/1</guid>
			<enclosure url="http://www.example.org/myaudiofile.mp3"
					length="12345"
					type="audio/mpeg" />
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Enclosures()) != 1 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}

	if feed.Entries[0].Enclosures()[0].URL != "http://www.example.org/myaudiofile.mp3" {
		t.Errorf("Incorrect enclosure URL, got: %s", feed.Entries[0].Enclosures()[0].URL)
	}

	if feed.Entries[0].Enclosures()[0].MimeType != "audio/mpeg" {
		t.Errorf("Incorrect enclosure type, got: %s", feed.Entries[0].Enclosures()[0].MimeType)
	}

	if feed.Entries[0].Enclosures()[0].Size != 12345 {
		t.Errorf("Incorrect enclosure length, got: %d", feed.Entries[0].Enclosures()[0].Size)
	}
}

func TestParseEntryWithIncorrectEnclosureLength(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
		<title>My Podcast Feed</title>
		<link>http://example.org</link>
		<author>some.email@example.org</author>
		<item>
			<title>Podcasting with RSS</title>
			<link>http://www.example.org/entries/1</link>
			<description>An overview of RSS podcasting</description>
			<pubDate>Fri, 15 Jul 2005 00:00:00 -0500</pubDate>
			<guid isPermaLink="true">http://www.example.org/entries/1</guid>
			<enclosure url="http://www.example.org/myaudiofile.mp3" length="invalid" type="audio/mpeg" />
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.EqualExportedValues(t, model.EnclosureList{
		{
			URL:      "http://www.example.org/myaudiofile.mp3",
			MimeType: "audio/mpeg",
		},
	}, feed.Entries[0].Enclosures())
}

func TestParseEntryWithDuplicatedEnclosureURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
		<title>My Podcast Feed</title>
		<link>http://example.org</link>
		<item>
			<title>Podcasting with RSS</title>
			<link>http://www.example.org/entries/1</link>
			<enclosure url="http://www.example.org/myaudiofile.mp3" type="audio/mpeg" />
			<enclosure url="   http://www.example.org/myaudiofile.mp3   " type="audio/mpeg" />
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Enclosures()) != 1 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}

	if feed.Entries[0].Enclosures()[0].URL != "http://www.example.org/myaudiofile.mp3" {
		t.Errorf("Incorrect enclosure URL, got: %s", feed.Entries[0].Enclosures()[0].URL)
	}
}

func TestParseEntryWithEmptyEnclosureURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
		<title>My Podcast Feed</title>
		<link>http://example.org</link>
		<author>some.email@example.org</author>
		<item>
			<title>Podcasting with RSS</title>
			<link>http://www.example.org/entries/1</link>
			<description>An overview of RSS podcasting</description>
			<pubDate>Fri, 15 Jul 2005 00:00:00 -0500</pubDate>
			<guid isPermaLink="true">http://www.example.org/entries/1</guid>
			<enclosure url=" " length="0"/>
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Enclosures()) != 0 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}
}

func TestParseEntryWithRelativeEnclosureURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
		<title>My Podcast Feed</title>
		<link>http://example.org</link>
		<author>some.email@example.org</author>
		<item>
			<title>Podcasting with RSS</title>
			<link>http://www.example.org/entries/1</link>
			<description>An overview of RSS podcasting</description>
			<pubDate>Fri, 15 Jul 2005 00:00:00 -0500</pubDate>
			<guid isPermaLink="true">http://www.example.org/entries/1</guid>
			<enclosure url=" /files/file.mp3  "/>
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if len(feed.Entries[0].Enclosures()) != 1 {
		t.Fatalf("Incorrect number of enclosures, got: %d", len(feed.Entries[0].Enclosures()))
	}

	if feed.Entries[0].Enclosures()[0].URL != "http://example.org/files/file.mp3" {
		t.Errorf("Incorrect enclosure URL, got: %q", feed.Entries[0].Enclosures()[0].URL)
	}
}

func TestParseEntryWithRelativeURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<link>https://example.org/</link>
			<item>
				<link>item.html</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].Title != "https://example.org/item.html" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}
}

func TestParseEntryWithCommentsURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>https://example.org/item1</link>
				<comments>
					https://example.org/comments
				</comments>
				<slash:comments>42</slash:comments>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].CommentsURL != "https://example.org/comments" {
		t.Errorf("Incorrect entry comments URL, got: %q", feed.Entries[0].CommentsURL)
	}
}

func TestParseEntryWithInvalidCommentsURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/</link>
			<item>
				<title>Item 1</title>
				<link>https://example.org/item1</link>
				<comments>
					:|Some text
				</comments>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Entries[0].CommentsURL != "" {
		t.Errorf("Incorrect entry comments URL, got: %q", feed.Entries[0].CommentsURL)
	}
}

func TestParseInvalidXml(t *testing.T) {
	data := `garbage`
	_, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err == nil {
		t.Error("Parse should returns an error")
	}
}

func TestParseFeedLinkWithInvalidCharacterEntity(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:slash="http://purl.org/rss/1.0/modules/slash/">
		<channel>
			<link>https://example.org/a&b</link>
			<title>Example Feed</title>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "https://example.org/a&b" {
		t.Errorf(`Incorrect url, got: %q`, feed.SiteURL)
	}
}

func TestParseEntryWithMediaGroup(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
		<channel>
		<title>My Example Feed</title>
		<link>https://example.org</link>
		<item>
			<title>Example Item</title>
			<link>http://www.example.org/entries/1</link>
			<enclosure type="application/x-bittorrent" url="https://example.org/file3.torrent" length="670053113">
			</enclosure>
			<media:group>
				<media:content type="application/x-bittorrent" url="https://example.org/file1.torrent"></media:content>
				<media:content type="application/x-bittorrent" url="https://example.org/file2.torrent" isDefault="true"></media:content>
				<media:content type="application/x-bittorrent" url="https://example.org/file3.torrent"></media:content>
				<media:content type="application/x-bittorrent" url="https://example.org/file4.torrent"></media:content>
				<media:content type="application/x-bittorrent" url="https://example.org/file4.torrent"></media:content>
				<media:content type="application/x-bittorrent" url=" file5.torrent  " fileSize="42"></media:content>
				<media:content type="application/x-bittorrent" url="  " fileSize="42"></media:content>
				<media:rating>nonadult</media:rating>
			</media:group>
			<media:thumbnail url="https://example.org/image.jpg" height="122" width="223"></media:thumbnail>
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.EqualExportedValues(t, model.EnclosureList{
		{
			URL:      "https://example.org/file3.torrent",
			MimeType: "application/x-bittorrent",
			Size:     670053113,
		},
		{
			URL:      "https://example.org/image.jpg",
			MimeType: "image/*",
		},
		{
			URL:      "https://example.org/file1.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://example.org/file2.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://example.org/file3.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://example.org/file4.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://example.org/file4.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://example.org/file5.torrent",
			MimeType: "application/x-bittorrent",
			Size:     42,
		},
	}, feed.Entries[0].Enclosures())
}

func TestParseEntryWithMediaContent(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
		<channel>
		<title>My Example Feed</title>
		<link>https://example.org</link>
		<item>
			<title>Example Item</title>
			<link>http://www.example.org/entries/1</link>
			<media:thumbnail url="https://example.org/thumbnail.jpg" />
			<media:thumbnail url="https://example.org/thumbnail.jpg" />
			<media:thumbnail url=" thumbnail.jpg  " />
			<media:thumbnail url="   " />
			<media:content url="https://example.org/media1.jpg" medium="image">
				<media:title type="html">Some Title for Media 1</media:title>
			</media:content>
			<media:content url="   /media2.jpg   " medium="image" />
			<media:content url="    " medium="image" />
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.EqualExportedValues(t, model.EnclosureList{
		{
			URL:      "https://example.org/thumbnail.jpg",
			MimeType: "image/*",
		},
		{
			URL:      "https://example.org/thumbnail.jpg",
			MimeType: "image/*",
		},
		{
			URL:      "https://example.org/thumbnail.jpg",
			MimeType: "image/*",
		},
		{
			URL:      "https://example.org/media1.jpg",
			MimeType: "image/*",
		},
		{
			URL:      "https://example.org/media2.jpg",
			MimeType: "image/*",
		},
	}, feed.Entries[0].Enclosures())
}

func TestParseEntryWithMediaPeerLink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
		<channel>
		<title>My Example Feed</title>
		<link>https://website.example.org</link>
		<item>
			<title>Example Item</title>
			<link>http://www.example.org/entries/1</link>
			<media:peerLink type="application/x-bittorrent" href="https://www.example.org/file.torrent" />
			<media:peerLink type="application/x-bittorrent" href="https://www.example.org/file.torrent" />
			<media:peerLink type="application/x-bittorrent" href="  file2.torrent   " />
			<media:peerLink type="application/x-bittorrent" href="    " />
		</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.EqualExportedValues(t, model.EnclosureList{
		{
			URL:      "https://www.example.org/file.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://www.example.org/file.torrent",
			MimeType: "application/x-bittorrent",
		},
		{
			URL:      "https://website.example.org/file2.torrent",
			MimeType: "application/x-bittorrent",
		},
	}, feed.Entries[0].Enclosures())
}

func TestParseItunesDuration(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Podcast Example</title>
			<link>http://www.example.com/index.html</link>
			<item>
				<title>Podcast Episode</title>
				<guid>http://example.com/episode.m4a</guid>
				<pubDate>Tue, 08 Mar 2016 12:00:00 GMT</pubDate>
				<itunes:duration>1:23:45</itunes:duration>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := 83
	result := feed.Entries[0].ReadingTime
	if expected != result {
		t.Errorf(`Unexpected podcast duration, got %d instead of %d`, result, expected)
	}
}

func TestParseIncorrectItunesDuration(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8"?>
		<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Podcast Example</title>
			<link>http://www.example.com/index.html</link>
			<item>
				<title>Podcast Episode</title>
				<guid>http://example.com/episode.m4a</guid>
				<pubDate>Tue, 08 Mar 2016 12:00:00 GMT</pubDate>
				<itunes:duration>invalid</itunes:duration>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	expected := 0
	result := feed.Entries[0].ReadingTime
	if expected != result {
		t.Errorf(`Unexpected podcast duration, got %d instead of %d`, result, expected)
	}
}

func TestEntryDescriptionFromItunesSummary(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Podcast Example</title>
			<link>http://www.example.com/index.html</link>
			<item>
				<title>Podcast Episode</title>
				<guid>http://example.com/episode.m4a</guid>
				<pubDate>Tue, 08 Mar 2016 12:00:00 GMT</pubDate>
				<itunes:subtitle>Episode Subtitle</itunes:subtitle>
				<itunes:summary>Episode Summary</itunes:summary>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	expected := "Episode Summary"
	result := feed.Entries[0].Content
	if expected != result {
		t.Errorf(`Unexpected podcast content, got %q instead of %q`, result, expected)
	}
}

func TestEntryDescriptionFromItunesSubtitle(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
		<channel>
			<title>Podcast Example</title>
			<link>http://www.example.com/index.html</link>
			<item>
				<title>Podcast Episode</title>
				<guid>http://example.com/episode.m4a</guid>
				<pubDate>Tue, 08 Mar 2016 12:00:00 GMT</pubDate>
				<itunes:subtitle>Episode Subtitle</itunes:subtitle>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	expected := "Episode Subtitle"
	result := feed.Entries[0].Content
	if expected != result {
		t.Errorf(`Unexpected podcast content, got %q instead of %q`, result, expected)
	}
}

func TestParseEntryWithRSSDescriptionAndMediaDescription(t *testing.T) {
	data := `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
		<channel>
			<title>Podcast Example</title>
			<link>http://www.example.com/index.html</link>
			<item>
				<title>Entry Title</title>
				<link>http://www.example.com/entries/1</link>
				<description>Entry Description</description>
				<media:description type="plain">Media Description</media:description>
			</item>
		</channel>
	</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	expected := "Entry Description"
	result := feed.Entries[0].Content
	if expected != result {
		t.Errorf(`Unexpected description, got %q instead of %q`, result, expected)
	}
}

func TestParseFeedWithCategories(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<category>Category 1</category>
			<category><![CDATA[Category 2]]></category>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries[0].Tags) != 2 {
		t.Errorf("Incorrect number of tags, got: %d", len(feed.Entries[0].Tags))
	}

	expected := []string{"Category 1", "Category 2"}
	result := feed.Entries[0].Tags

	for i, tag := range result {
		if tag != expected[i] {
			t.Errorf("Incorrect tag, got: %q", tag)
		}
	}
}

func TestParseEntryWithCategories(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<category>Category 3</category>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<category>Category 1</category>
				<category><![CDATA[Category 2]]></category>
				<category>Category 2</category>
				<category>Category 0</category>
				<category>   </category>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, []string{"Category 0", "Category 1", "Category 2"},
		feed.Entries[0].Tags)
}

func TestParseFeedWithItunesCategories(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<itunes:category text="Society &amp; Culture">
				<itunes:category text="Documentary" />
			</itunes:category>
			<itunes:category text="Health">
				<itunes:category text="Mental Health" />
			</itunes:category>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries[0].Tags) != 4 {
		t.Errorf("Incorrect number of tags, got: %d", len(feed.Entries[0].Tags))
	}

	expected := []string{"Documentary", "Health", "Mental Health", "Society & Culture"}
	result := feed.Entries[0].Tags

	for i, tag := range result {
		if tag != expected[i] {
			t.Errorf("Incorrect tag, got: %q", tag)
		}
	}
}

func TestParseEntryWithMediaCategories(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
				<media:category label="Visual Art">visual_art</media:category>
				<media:category scheme="http://search.yahoo.com/mrss/category_ schema">music/artist/album/song</media:category>
				<media:category scheme="urn:flickr:tags">ycantpark mobile</media:category>
				<media:category scheme="http://dmoz.org" label="Ace Ventura - Pet Detective">Arts/Movies/Titles/A/Ace_Ventura_Series/Ace_Ventura_ -_Pet_Detective</media:category>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	require.Len(t, feed.Entries, 1)
	assert.Equal(t, []string{"Ace Ventura - Pet Detective", "Visual Art"},
		feed.Entries[0].Tags)
}

func TestParseFeedWithTTLField(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<ttl>60</ttl>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.TTL != 60 {
		t.Errorf("Incorrect TTL, got: %d", feed.TTL)
	}
}

func TestParseFeedWithIncorrectTTLValue(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rss version="2.0">
		<channel>
			<title>Example</title>
			<link>https://example.org/</link>
			<ttl>invalid</ttl>
			<item>
				<title>Test</title>
				<link>https://example.org/item</link>
			</item>
		</channel>
		</rss>`

	feed, err := parser.ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.TTL != 0 {
		t.Errorf("Incorrect TTL, got: %d", feed.TTL)
	}
}
