// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package opml // import "miniflux.app/v2/internal/reader/opml"

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Specs: http://opml.org/spec2.opml
type opmlDocument struct {
	XMLName  xml.Name              `xml:"opml"`
	Version  string                `xml:"version,attr"`
	Header   opmlHeader            `xml:"head"`
	Outlines opmlOutlineCollection `xml:"body>outline"`
}

func NewOPMLDocument() *opmlDocument {
	return &opmlDocument{}
}

type opmlHeader struct {
	Title       string `xml:"title,omitempty"`
	DateCreated string `xml:"dateCreated,omitempty"`
	OwnerName   string `xml:"ownerName,omitempty"`
}

type opmlOutline struct {
	Title       string                `xml:"title,attr,omitempty"`
	Text        string                `xml:"text,attr"`
	FeedURL     string                `xml:"xmlUrl,attr,omitempty"`
	SiteURL     string                `xml:"htmlUrl,attr,omitempty"`
	Description string                `xml:"description,attr,omitempty"`
	Outlines    opmlOutlineCollection `xml:"outline,omitempty"`
}

func (o opmlOutline) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type opmlOutlineXml opmlOutline

	outlineType := ""
	if o.IsSubscription() {
		outlineType = "rss"
	}

	err := e.EncodeElement(struct {
		opmlOutlineXml

		Type string `xml:"type,attr,omitempty"`
	}{
		opmlOutlineXml: opmlOutlineXml(o),
		Type:           outlineType,
	}, start)
	if err != nil {
		return fmt.Errorf("reader/opml: %w", err)
	}
	return nil
}

func (o opmlOutline) IsSubscription() bool {
	return strings.TrimSpace(o.FeedURL) != ""
}

func (o opmlOutline) GetTitle() string {
	if o.Title != "" {
		return o.Title
	}

	if o.Text != "" {
		return o.Text
	}

	if o.SiteURL != "" {
		return o.SiteURL
	}

	if o.FeedURL != "" {
		return o.FeedURL
	}

	return ""
}

func (o opmlOutline) GetSiteURL() string {
	if o.SiteURL != "" {
		return o.SiteURL
	}

	return o.FeedURL
}

type opmlOutlineCollection []opmlOutline

func (o opmlOutlineCollection) HasChildren() bool {
	return len(o) > 0
}
