// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package rdf // import "miniflux.app/v2/internal/reader/rdf"

import (
	"encoding/xml"

	"miniflux.app/v2/internal/reader/dublincore"
)

// rdf sepcs: https://web.resource.org/rss/1.0/spec
type rdf struct {
	XMLName xml.Name   `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF"`
	Channel rdfChannel `xml:"channel"`
	Items   []rdfItem  `xml:"item"`
}

type rdfChannel struct {
	dublincore.DublinCoreChannelElement

	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type rdfItem struct {
	dublincore.DublinCoreItemElement

	Title       string `xml:"http://purl.org/rss/1.0/ title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}
