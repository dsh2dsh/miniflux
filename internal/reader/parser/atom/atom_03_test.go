// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package atom_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/reader/parser"
)

func TestParseAtom03(t *testing.T) {
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

	feed, err := parser.ParseBytes("http://diveintomark.org/atom.xml",
		[]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "dive into mark" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}

	if feed.FeedURL != "http://diveintomark.org/atom.xml" {
		t.Errorf("Incorrect feed URL, got: %s", feed.FeedURL)
	}

	if feed.SiteURL != "http://diveintomark.org/" {
		t.Errorf("Incorrect site URL, got: %s", feed.SiteURL)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	tz := time.FixedZone("Test Case Time", -int((4 * time.Hour).Seconds()))
	if !feed.Entries[0].Date.Equal(time.Date(2003, time.December, 13, 8, 29, 29, 0, tz)) {
		t.Errorf("Incorrect entry date, got: %v", feed.Entries[0].Date)
	}

	if feed.Entries[0].Hash != "3d2e27978dad4dca" {
		t.Errorf("Incorrect entry hash, got: %s", feed.Entries[0].Hash)
	}

	if feed.Entries[0].URL != "http://diveintomark.org/2003/12/13/atom03" {
		t.Errorf("Incorrect entry URL, got: %s", feed.Entries[0].URL)
	}

	if feed.Entries[0].Title != "Atom 0.3 snapshot" {
		t.Errorf("Incorrect entry title, got: %s", feed.Entries[0].Title)
	}

	if feed.Entries[0].Content != "<p>HTML content</p>" {
		t.Errorf("Incorrect entry content, got: %s", feed.Entries[0].Content)
	}

	if feed.Entries[0].Author != "Mark Pilgrim" {
		t.Errorf("Incorrect entry author, got: %s", feed.Entries[0].Author)
	}
}

func TestParseAtom03WithoutSiteURL(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed version="0.3" xmlns="http://purl.org/atom/ns#">
		<modified>2003-12-13T18:30:02Z</modified>
		<author><name>Mark Pilgrim</name></author>
		<entry>
			<title>Atom 0.3 snapshot</title>
			<link rel="alternate" type="text/html" href="http://diveintomark.org/2003/12/13/atom03"/>
			<id>tag:diveintomark.org,2003:3.2397</id>
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/atom.xml",
		[]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.SiteURL != "http://diveintomark.org/atom.xml" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseAtom03WithoutFeedTitle(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed version="0.3" xmlns="http://purl.org/atom/ns#">
		<link rel="alternate" type="text/html" href="http://diveintomark.org/"/>
		<modified>2003-12-13T18:30:02Z</modified>
		<author><name>Mark Pilgrim</name></author>
		<entry>
			<title>Atom 0.3 snapshot</title>
			<link rel="alternate" type="text/html" href="http://diveintomark.org/2003/12/13/atom03"/>
			<id>tag:diveintomark.org,2003:3.2397</id>
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if feed.Title != "http://diveintomark.org/" {
		t.Errorf("Incorrect title, got: %s", feed.Title)
	}
}

func TestParseAtom03WithoutEntryTitleButWithLink(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<feed version="0.3" xmlns="http://purl.org/atom/ns#">
		<title>dive into mark</title>
		<link rel="alternate" type="text/html" href="http://diveintomark.org/"/>
		<modified>2003-12-13T18:30:02Z</modified>
		<author><name>Mark Pilgrim</name></author>
		<entry>
			<link rel="alternate" type="text/html" href="http://diveintomark.org/2003/12/13/atom03"/>
			<id>tag:diveintomark.org,2003:3.2397</id>
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)
	require.NotEmpty(t, feed.Entries)
	assert.Empty(t, feed.Entries[0].Title)
	assert.Equal(t, "http://diveintomark.org/2003/12/13/atom03",
		feed.Entries[0].URL)
}

func TestParseAtom03WithSummaryOnly(t *testing.T) {
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
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/", []byte(data))
	require.NoError(t, err)
	require.NotNil(t, feed)

	assert.Len(t, feed.Entries, 1)
	assert.Equal(t, "It's a test", feed.Entries[0].Content)
}

func TestParseAtom03WithXMLContent(t *testing.T) {
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
			<content mode="xml" type="text/html"><p>Some text.</p></content>
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Content != "<p>Some text.</p>" {
		t.Errorf("Incorrect entry content, got: %s", feed.Entries[0].Content)
	}
}

func TestParseAtom03WithBase64Content(t *testing.T) {
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
			<content mode="base64" type="text/html">PHA+U29tZSB0ZXh0LjwvcD4=</content>
		</entry>
	</feed>`

	feed, err := parser.ParseBytes("http://diveintomark.org/", []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Incorrect number of entries, got: %d", len(feed.Entries))
	}

	if feed.Entries[0].Content != "<p>Some text.</p>" {
		t.Errorf("Incorrect entry content, got: %s", feed.Entries[0].Content)
	}
}
