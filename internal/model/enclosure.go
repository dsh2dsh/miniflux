// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"fmt"
	"iter"
	"net/url"
	"strings"
)

// Enclosure represents an attachment.
type Enclosure struct {
	URL              string `json:"url,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
	Size             int64  `json:"size,omitempty"`
	MediaProgression int64  `json:"media_progression,omitempty"`
	Height           int    `json:"height,omitempty"`
	Width            int    `json:"width,omitempty"`

	originalURL string
	parsedURL   *url.URL
}

type EnclosureUpdateRequest struct {
	MediaProgression int64 `json:"media_progression,omitempty"`
}

func (self *Enclosure) WithURL(u *url.URL) *Enclosure {
	if u != nil {
		self.parsedURL, self.URL = u, u.String()
	} else {
		self.WithURLString("")
	}
	return self
}

func (self *Enclosure) WithURLString(urlStr string) *Enclosure {
	if self.URL != urlStr {
		self.parsedURL, self.URL = nil, urlStr
	}
	return self
}

func (self *Enclosure) ParsedURL() (*url.URL, error) {
	if self.parsedURL != nil {
		return self.parsedURL, nil
	}

	u, err := url.Parse(self.URL)
	if err != nil {
		return nil, fmt.Errorf("parse enclosure URL: %w", err)
	}
	self.parsedURL = u
	return u, nil
}

// Html5MimeType will modify the actual MimeType to allow direct playback from HTML5 player for some kind of MimeType
func (self *Enclosure) Html5MimeType() string {
	if self.MimeType == "video/m4v" {
		return "video/x-m4v"
	}
	return self.MimeType
}

func (self *Enclosure) IsAudio() bool {
	return strings.HasPrefix(strings.ToLower(self.MimeType), "audio/")
}

func (self *Enclosure) IsVideo() bool {
	return strings.HasPrefix(strings.ToLower(self.MimeType), "video/")
}

func (self *Enclosure) IsImage() bool {
	mimeType := strings.ToLower(self.MimeType)
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	mediaURL := strings.ToLower(self.URL)
	return strings.HasSuffix(mediaURL, ".jpg") ||
		strings.HasSuffix(mediaURL, ".jpeg") ||
		strings.HasSuffix(mediaURL, ".png") ||
		strings.HasSuffix(mediaURL, ".gif")
}

func (self *Enclosure) ReplaceURL(urlStr string) string {
	self.originalURL = self.URL
	return self.WithURLString(urlStr).originalURL
}

func (self *Enclosure) OriginalURL() string {
	if self.originalURL != "" {
		return self.originalURL
	}
	return self.URL
}

// EnclosureList represents a list of attachments.
type EnclosureList []Enclosure

// FindMediaPlayerEnclosure returns the first enclosure that can be played by a media player.
func (self EnclosureList) FindMediaPlayerEnclosure() *Enclosure {
	for i := range self {
		enclosure := &self[i]
		if enclosure.URL != "" {
			if enclosure.IsAudio() || enclosure.IsVideo() {
				return enclosure
			}
		}
	}
	return nil
}

func (self EnclosureList) ContainsAudioOrVideo() bool {
	for i := range self {
		enclosure := &self[i]
		if enclosure.IsAudio() || enclosure.IsVideo() {
			return true
		}
	}
	return false
}

func (self EnclosureList) WithMimeType(mimeType string) iter.Seq[*Enclosure] {
	return func(yield func(*Enclosure) bool) {
		for i := range self {
			enc := &self[i]
			if strings.EqualFold(enc.MimeType, mimeType) {
				if !yield(enc) {
					return
				}
			}
		}
	}
}
