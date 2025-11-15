// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package parser // import "miniflux.app/v2/internal/reader/parser"

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkParseFeed(b *testing.B) {
	tests := []struct {
		name    string
		baseURL string
		feed    []byte
	}{
		{
			name:    "large_atom.xml",
			baseURL: "https://dustri.org/b",
		},
		{
			name:    "large_rss.xml",
			baseURL: "https://dustri.org/b",
		},
		{
			name:    "small_atom.xml",
			baseURL: "https://github.com/miniflux/v2/commits/main",
		},
	}

	for i := range tests {
		tt := &tests[i]
		data, err := os.ReadFile("testdata/" + tt.name)
		require.NoError(b, err)
		tt.feed = data
	}

	b.ReportAllocs()
	for b.Loop() {
		for i := range tests {
			tt := &tests[i]
			ParseFeed(tt.baseURL, bytes.NewReader(tt.feed))
		}
	}
}

func BenchmarkParseBytes(b *testing.B) {
	tests := []struct {
		name    string
		baseURL string
		feed    []byte
	}{
		{
			name:    "large_atom.xml",
			baseURL: "https://dustri.org/b",
		},
		{
			name:    "large_rss.xml",
			baseURL: "https://dustri.org/b",
		},
		{
			name:    "small_atom.xml",
			baseURL: "https://github.com/miniflux/v2/commits/main",
		},
	}

	for i := range tests {
		tt := &tests[i]
		data, err := os.ReadFile("testdata/" + tt.name)
		require.NoError(b, err)
		tt.feed = data
	}

	b.ReportAllocs()
	for b.Loop() {
		for i := range tests {
			tt := &tests[i]
			ParseBytes(tt.baseURL, tt.feed)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add("https://z.org", `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Example Feed</title>
<link href="http://z.org/"/>
<link href="/k"/>
<updated>2003-12-13T18:30:02Z</updated>
<author><name>John Doe</name></author>
<id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>
<entry>
<title>a</title>
<link href="http://example.org/b"/>
<id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
<updated>2003-12-13T18:30:02Z</updated>
<summary>c</summary>
</entry>
</feed>`)
	f.Add("https://z.org", `<?xml version="1.0"?>
<rss version="2.0">
<channel>
<title>a</title>
<link>http://z.org</link>
<item>
<title>a</title>
<link>http://z.org</link>
<description>d</description>
<pubDate>Tue, 03 Jun 2003 09:39:21 GMT</pubDate>
<guid>l</guid>
</item>
</channel>
</rss>`)
	f.Add("https://z.org", `<?xml version="1.0" encoding="utf-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/">
<channel>
<title>a</title>
<link>http://z.org/</link>
</channel>
<item>
<title>a</title>
<link>/</link>
<description>c</description>
</item>
</rdf:RDF>`)
	f.Add("http://z.org", `{
"version": "http://jsonfeed.org/version/1",
"title": "a",
"home_page_url": "http://z.org/",
"feed_url": "http://z.org/a.json",
"items": [
{"id": "2","content_text": "a","url": "https://z.org/2"},
{"id": "1","content_html": "<a","url":"http://z.org/1"}]}`)
	f.Fuzz(func(t *testing.T, url, data string) {
		ParseBytes(url, []byte(data))
	})
}

func TestParseAtom03Feed(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed version="0.3" xmlns="http://purl.org/atom/ns#">
		<title>dive into mark</title>
		<link rel="alternate" type="text/html" href="http://diveintomark.org/"/>
		<modified>2003-12-13T18:30:02Z</modified>
		<author><name>Mark Pilgrim</name></author>
		<entry>
			<title>Atom 0.3 snapshot</title>
			<link rel="alternate" type="text/html" href="http://diveintomark.org/2003/12/13/atom03"/>
			<id>tag:diveintomark.org,2003:3.2397</id>
			<issued>2003-12-13T08:29:29-04:00</issued>
			<modified>2003-12-13T18:30:02Z</modified>
			<summary type="text/plain">It&apos;s a test</summary>
			<content type="text/html" mode="escaped"><![CDATA[<p>HTML content</p>]]></content>
		</entry>
	</feed>`

	feed, err := ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "dive into mark" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseAtom10Feed(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">

	  <title>Example Feed</title>
	  <link href="http://example.org/"/>
	  <updated>2003-12-13T18:30:02Z</updated>
	  <author>
		<name>John Doe</name>
	  </author>
	  <id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>

	  <entry>
		<title>Atom-Powered Robots Run Amok</title>
		<link href="http://example.org/2003/12/13/atom03"/>
		<id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
		<updated>2003-12-13T18:30:02Z</updated>
		<summary>Some text.</summary>
	  </entry>

	</feed>`

	feed, err := ParseBytes("https://example.org/", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "Example Feed" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseAtomFeedWithRelativeURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
	  <title>Example Feed</title>
	  <link href="/blog/atom.xml" rel="self" type="application/atom+xml"/>
	  <link href="/blog"/>

	  <entry>
		<title>Test</title>
		<link href="/blog/article.html"/>
		<link href="/blog/article.html" rel="alternate" type="text/html"/>
		<id>/blog/article.html</id>
		<updated>2003-12-13T18:30:02Z</updated>
		<summary>Some text.</summary>
	  </entry>

	</feed>`

	feed, err := ParseBytes("https://example.org/blog/atom.xml", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.FeedURL != "https://example.org/blog/atom.xml" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "https://example.org/blog" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if feed.Entries[0].URL != "https://example.org/blog/article.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseRSS(t *testing.T) {
	data := `<?xml version="1.0"?>
	<rss version="2.0">
	<channel>
		<title>Liftoff News</title>
		<link>http://liftoff.msfc.nasa.gov/</link>
		<item>
			<title>Star City</title>
			<link>http://liftoff.msfc.nasa.gov/news/2003/news-starcity.asp</link>
			<description>How do Americans get ready to work with Russians aboard the International Space Station? They take a crash course in culture, language and protocol at Russia's &lt;a href="http://howe.iki.rssi.ru/GCTC/gctc_e.htm"&gt;Star City&lt;/a&gt;.</description>
			<pubDate>Tue, 03 Jun 2003 09:39:21 GMT</pubDate>
			<guid>http://liftoff.msfc.nasa.gov/2003/06/03.html#item573</guid>
		</item>
	</channel>
	</rss>`

	feed, err := ParseBytes("http://liftoff.msfc.nasa.gov/", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "Liftoff News" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseRSSFeedWithRelativeURL(t *testing.T) {
	data := `<?xml version="1.0"?>
	<rss version="2.0">
	<channel>
		<title>Example Feed</title>
		<link>/blog</link>
		<item>
			<title>Example Entry</title>
			<link>/blog/article.html</link>
			<description>Something</description>
			<pubDate>Tue, 03 Jun 2003 09:39:21 GMT</pubDate>
			<guid>1234</guid>
		</item>
	</channel>
	</rss>`

	feed, err := ParseBytes("http://example.org/rss.xml", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "Example Feed" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}

	if feed.FeedURL != "http://example.org/rss.xml" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "http://example.org/blog" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if feed.Entries[0].URL != "http://example.org/blog/article.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseRDF(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rdf:RDF
		  xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
		  xmlns="http://purl.org/rss/1.0/"
		>

		  <channel>
			<title>RDF Example</title>
			<link>http://example.org/</link>
		  </channel>

		  <item>
			<title>Title</title>
			<link>http://example.org/item</link>
			<description>Test</description>
		  </item>
		</rdf:RDF>`

	feed, err := ParseBytes("http://example.org/", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "RDF Example" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseRDFWithRelativeURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
		<rdf:RDF
		  xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
		  xmlns="http://purl.org/rss/1.0/"
		>

		  <channel>
			<title>RDF Example</title>
			<link>/blog</link>
		  </channel>

		  <item>
			<title>Title</title>
			<link>/blog/article.html</link>
			<description>Test</description>
		  </item>
		</rdf:RDF>`

	feed, err := ParseBytes("http://example.org/rdf.xml", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.FeedURL != "http://example.org/rdf.xml" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "http://example.org/blog" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if feed.Entries[0].URL != "http://example.org/blog/article.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseJson(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
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

	feed, err := ParseBytes("https://example.org/feed.json", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "My Example Feed" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseJsonFeedWithRelativeURL(t *testing.T) {
	data := `{
		"version": "https://jsonfeed.org/version/1",
		"title": "My Example Feed",
		"home_page_url": "/blog",
		"feed_url": "/blog/feed.json",
		"items": [
			{
				"id": "2",
				"content_text": "This is a second item.",
				"url": "/blog/article.html"
			}
		]
	}`

	feed, err := ParseBytes("https://example.org/blog/feed.json", []byte(data))
	if err != nil {
		t.Error(err)
	}

	if feed.Title != "My Example Feed" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}

	if feed.FeedURL != "https://example.org/blog/feed.json" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "https://example.org/blog" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if feed.Entries[0].URL != "https://example.org/blog/article.html" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}
}

func TestParseUnknownFeed(t *testing.T) {
	data := `
		<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
		<html xmlns="http://www.w3.org/1999/xhtml">
			<head>
				<title>Title of document</title>
			</head>
			<body>
				some content
			</body>
		</html>
	`

	_, err := ParseBytes("https://example.org/", []byte(data))
	if err == nil {
		t.Error("ParseFeed must returns an error")
	}
}

func TestParseEmptyFeed(t *testing.T) {
	_, err := ParseBytes("", []byte{})
	if err == nil {
		t.Error("ParseFeed must returns an error")
	}
}

func TestParseEncodings(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "encoding_ISO-8859-1"},
		{name: "encoding_WINDOWS-1251"},
		{
			name:    "no_encoding_ISO-8859-1",
			wantErr: true,
		},
		{name: "rdf_UTF8"},
		{name: "urdu_UTF8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile("testdata/" + tt.name + ".xml")
			require.NoError(t, err)

			_, err = ParseBytes("", b)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
