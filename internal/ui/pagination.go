// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"html/template"
	"net/url"
	"strconv"
)

type pagination struct {
	Route        string
	Total        int
	Offset       int
	ItemsPerPage int
	ShowNext     bool
	ShowLast     bool
	ShowFirst    bool
	ShowPrev     bool
	NextOffset   int
	LastOffset   int
	PrevOffset   int
	FirstOffset  int
	SearchQuery  string
}

func getPagination(route string, total, offset, itemsPerPage int) *pagination {
	var nextOffset, prevOffset, firstOffset int
	lastOffset := (total / itemsPerPage) * itemsPerPage
	if lastOffset == total {
		lastOffset -= itemsPerPage
	}

	showNext := (total - offset) > itemsPerPage
	showPrev := offset > 0
	showLast := showNext
	showFirst := showPrev

	if showNext {
		nextOffset = offset + itemsPerPage
	}

	if showPrev {
		prevOffset = offset - itemsPerPage
	}

	return &pagination{
		Route:        route,
		Total:        total,
		Offset:       offset,
		ItemsPerPage: itemsPerPage,
		ShowNext:     showNext,
		ShowLast:     showLast,
		NextOffset:   nextOffset,
		LastOffset:   lastOffset,
		ShowPrev:     showPrev,
		ShowFirst:    showFirst,
		PrevOffset:   prevOffset,
		FirstOffset:  firstOffset,
	}
}

func (self *pagination) FirstDisabled() template.HTMLAttr {
	return disabled(self.ShowFirst)
}

func disabled(enabled bool) template.HTMLAttr {
	if enabled {
		return ""
	}
	return "disabled"
}

func (self *pagination) FirstRoute() template.URL {
	return self.offsetRoute(self.FirstOffset)
}

func (self *pagination) offsetRoute(offset int) template.URL {
	if q := self.offsetQuery(offset); q != "" {
		return template.URL(self.Route + "?" + q)
	}
	return template.URL(self.Route)
}

func (self *pagination) offsetQuery(offset int) string {
	v := url.Values{}
	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}
	if self.SearchQuery != "" {
		v.Set("q", url.QueryEscape(self.SearchQuery))
	}
	return v.Encode()
}

func (self *pagination) PrevDisabled() template.HTMLAttr {
	return disabled(self.ShowPrev)
}

func (self *pagination) PrevRoute() template.URL {
	return self.offsetRoute(self.PrevOffset)
}

func (self *pagination) NextDisabled() template.HTMLAttr {
	return disabled(self.ShowNext)
}

func (self *pagination) NextRoute() template.URL {
	return self.offsetRoute(self.NextOffset)
}

func (self *pagination) LastDisabled() template.HTMLAttr {
	return disabled(self.ShowLast)
}

func (self *pagination) LastRoute() template.URL {
	return self.offsetRoute(self.LastOffset)
}
