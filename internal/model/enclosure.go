// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model // import "miniflux.app/v2/internal/model"

import (
	"strings"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/http/mux"
	"miniflux.app/v2/internal/mediaproxy"
	"miniflux.app/v2/internal/urllib"
)

// Enclosure represents an attachment.
type Enclosure struct {
	URL              string `json:"url,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
	Size             int64  `json:"size,omitempty"`
	MediaProgression int64  `json:"media_progression,omitempty"`
}

type EnclosureUpdateRequest struct {
	MediaProgression int64 `json:"media_progression,omitempty"`
}

// Html5MimeType will modify the actual MimeType to allow direct playback from HTML5 player for some kind of MimeType
func (e *Enclosure) Html5MimeType() string {
	if e.MimeType == "video/m4v" {
		return "video/x-m4v"
	}
	return e.MimeType
}

func (e *Enclosure) IsAudio() bool {
	return strings.HasPrefix(strings.ToLower(e.MimeType), "audio/")
}

func (e *Enclosure) IsVideo() bool {
	return strings.HasPrefix(strings.ToLower(e.MimeType), "video/")
}

func (e *Enclosure) IsImage() bool {
	mimeType := strings.ToLower(e.MimeType)
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	mediaURL := strings.ToLower(e.URL)
	return strings.HasSuffix(mediaURL, ".jpg") ||
		strings.HasSuffix(mediaURL, ".jpeg") ||
		strings.HasSuffix(mediaURL, ".png") ||
		strings.HasSuffix(mediaURL, ".gif")
}

func (e *Enclosure) ProxifyEnclosureURL(router *mux.ServeMux) {
	proxyOption := config.Opts.MediaProxyMode()

	if proxyOption == "all" || proxyOption != "none" && !urllib.IsHTTPS(e.URL) {
		proxifyAbsoluteURLIfMimeType(e, router)
	}
}

func proxifyAbsoluteURLIfMimeType(e *Enclosure, router *mux.ServeMux) {
	for _, mediaType := range config.Opts.MediaProxyResourceTypes() {
		if strings.HasPrefix(e.MimeType, mediaType+"/") {
			e.URL = mediaproxy.ProxifyAbsoluteURL(router, e.URL)
			break
		}
	}
}

// EnclosureList represents a list of attachments.
type EnclosureList []Enclosure

func (el EnclosureList) ContainsAudioOrVideo() bool {
	for i := range el {
		enclosure := &el[i]
		if enclosure.IsAudio() || enclosure.IsVideo() {
			return true
		}
	}
	return false
}

func (el EnclosureList) ProxifyEnclosureURL(router *mux.ServeMux) {
	proxyOption := config.Opts.MediaProxyMode()

	if proxyOption != "none" {
		for i := range el {
			if urllib.IsHTTPS(el[i].URL) {
				proxifyAbsoluteURLIfMimeType(&el[i], router)
			}
		}
	}
}

// FindMediaPlayerEnclosure returns the first enclosure that can be played by a media player.
func (el EnclosureList) FindMediaPlayerEnclosure() *Enclosure {
	for i := range el {
		enclosure := &el[i]
		if enclosure.URL != "" {
			if enclosure.IsAudio() || enclosure.IsVideo() {
				return enclosure
			}
		}
	}
	return nil
}
