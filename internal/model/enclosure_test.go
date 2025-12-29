// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"
)

func TestEnclosure_Html5MimeTypeGivesOriginalMimeType(t *testing.T) {
	enclosure := Enclosure{MimeType: "thing/thisMimeTypeIsNotExpectedToBeReplaced"}
	if enclosure.Html5MimeType() != enclosure.MimeType {
		t.Fatalf(
			"HTML5 MimeType must provide original MimeType if not explicitly Replaced. Got %s ,expected '%s' ",
			enclosure.Html5MimeType(),
			enclosure.MimeType,
		)
	}
}

func TestEnclosure_Html5MimeTypeReplaceStandardM4vByAppleSpecificMimeType(t *testing.T) {
	enclosure := Enclosure{MimeType: "video/m4v"}
	if enclosure.Html5MimeType() != "video/x-m4v" {
		// Solution from this stackoverflow discussion:
		// https://stackoverflow.com/questions/15277147/m4v-mimetype-video-mp4-or-video-m4v/66945470#66945470
		// tested at the time of this commit (06/2023) on latest Firefox & Vivaldi on this feed
		// https://www.florenceporcel.com/podcast/lfhdu.xml
		t.Fatalf(
			"HTML5 MimeType must be replaced by 'video/x-m4v' when originally video/m4v to ensure playbacks in browsers. Got '%s'",
			enclosure.Html5MimeType(),
		)
	}
}

func TestEnclosure_IsAudio(t *testing.T) {
	testCases := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"MP3 audio", "audio/mpeg", true},
		{"WAV audio", "audio/wav", true},
		{"OGG audio", "audio/ogg", true},
		{"Mixed case audio", "Audio/MP3", true},
		{"Video file", "video/mp4", false},
		{"Image file", "image/jpeg", false},
		{"Text file", "text/plain", false},
		{"Empty mime type", "", false},
		{"Audio with extra info", "audio/mpeg; charset=utf-8", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enclosure := &Enclosure{MimeType: tc.mimeType}
			if got := enclosure.IsAudio(); got != tc.expected {
				t.Errorf("IsAudio() = %v, want %v for mime type %s", got, tc.expected, tc.mimeType)
			}
		})
	}
}

func TestEnclosure_IsVideo(t *testing.T) {
	testCases := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"MP4 video", "video/mp4", true},
		{"AVI video", "video/avi", true},
		{"WebM video", "video/webm", true},
		{"M4V video", "video/m4v", true},
		{"Mixed case video", "Video/MP4", true},
		{"Audio file", "audio/mpeg", false},
		{"Image file", "image/jpeg", false},
		{"Text file", "text/plain", false},
		{"Empty mime type", "", false},
		{"Video with extra info", "video/mp4; codecs=\"avc1.42E01E\"", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enclosure := &Enclosure{MimeType: tc.mimeType}
			if got := enclosure.IsVideo(); got != tc.expected {
				t.Errorf("IsVideo() = %v, want %v for mime type %s", got, tc.expected, tc.mimeType)
			}
		})
	}
}

func TestEnclosure_IsImage(t *testing.T) {
	testCases := []struct {
		name     string
		mimeType string
		url      string
		expected bool
	}{
		{"JPEG image by mime", "image/jpeg", "http://example.com/file", true},
		{"PNG image by mime", "image/png", "http://example.com/file", true},
		{"GIF image by mime", "image/gif", "http://example.com/file", true},
		{"Mixed case image mime", "Image/JPEG", "http://example.com/file", true},
		{"JPG file extension", "application/octet-stream", "http://example.com/photo.jpg", true},
		{"JPEG file extension", "text/plain", "http://example.com/photo.jpeg", true},
		{"PNG file extension", "unknown/type", "http://example.com/photo.png", true},
		{"GIF file extension", "binary/data", "http://example.com/photo.gif", true},
		{"Mixed case extension", "text/plain", "http://example.com/photo.JPG", true},
		{"Image mime and extension", "image/jpeg", "http://example.com/photo.jpg", true},
		{"Video file", "video/mp4", "http://example.com/video.mp4", false},
		{"Audio file", "audio/mpeg", "http://example.com/audio.mp3", false},
		{"Text file", "text/plain", "http://example.com/file.txt", false},
		{"No extension", "text/plain", "http://example.com/file", false},
		{"Other extension", "text/plain", "http://example.com/file.pdf", false},
		{"Empty values", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enclosure := &Enclosure{MimeType: tc.mimeType, URL: tc.url}
			if got := enclosure.IsImage(); got != tc.expected {
				t.Errorf("IsImage() = %v, want %v for mime type %s and URL %s", got, tc.expected, tc.mimeType, tc.url)
			}
		})
	}
}

func TestEnclosureList_FindMediaPlayerEnclosure(t *testing.T) {
	testCases := []struct {
		name        string
		enclosures  EnclosureList
		expectedNil bool
	}{
		{
			name: "Returns first audio enclosure",
			enclosures: EnclosureList{
				Enclosure{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
				Enclosure{URL: "http://example.com/video.mp4", MimeType: "video/mp4"},
			},
			expectedNil: false,
		},
		{
			name: "Returns first video enclosure",
			enclosures: EnclosureList{
				Enclosure{URL: "http://example.com/video.mp4", MimeType: "video/mp4"},
				Enclosure{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
			},
			expectedNil: false,
		},
		{
			name: "Skips image enclosure and returns audio",
			enclosures: EnclosureList{
				Enclosure{URL: "http://example.com/image.jpg", MimeType: "image/jpeg"},
				Enclosure{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
			},
			expectedNil: false,
		},
		{
			name: "Skips enclosure with empty URL",
			enclosures: EnclosureList{
				Enclosure{URL: "", MimeType: "audio/mpeg"},
				Enclosure{URL: "http://example.com/audio.mp3", MimeType: "audio/mpeg"},
			},
			expectedNil: false,
		},
		{
			name: "Returns nil for no media enclosures",
			enclosures: EnclosureList{
				Enclosure{URL: "http://example.com/image.jpg", MimeType: "image/jpeg"},
				Enclosure{URL: "http://example.com/doc.pdf", MimeType: "application/pdf"},
			},
			expectedNil: true,
		},
		{
			name:        "Returns nil for empty list",
			enclosures:  EnclosureList{},
			expectedNil: true,
		},
		{
			name: "Returns nil for all empty URLs",
			enclosures: EnclosureList{
				Enclosure{URL: "", MimeType: "audio/mpeg"},
				Enclosure{URL: "", MimeType: "video/mp4"},
			},
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.enclosures.FindMediaPlayerEnclosure()
			if tc.expectedNil {
				if result != nil {
					t.Errorf("FindMediaPlayerEnclosure() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("FindMediaPlayerEnclosure() = nil, want non-nil")
				} else if !result.IsAudio() && !result.IsVideo() {
					t.Errorf("FindMediaPlayerEnclosure() returned non-media enclosure: %s", result.MimeType)
				}
			}
		})
	}
}

func TestEnclosureList_ContainsAudioOrVideo(t *testing.T) {
	testCases := []struct {
		name       string
		enclosures EnclosureList
		expected   bool
	}{
		{
			name: "Contains audio",
			enclosures: EnclosureList{
				Enclosure{MimeType: "audio/mpeg"},
				Enclosure{MimeType: "image/jpeg"},
			},
			expected: true,
		},
		{
			name: "Contains video",
			enclosures: EnclosureList{
				Enclosure{MimeType: "image/jpeg"},
				Enclosure{MimeType: "video/mp4"},
			},
			expected: true,
		},
		{
			name: "Contains both audio and video",
			enclosures: EnclosureList{
				Enclosure{MimeType: "audio/mpeg"},
				Enclosure{MimeType: "video/mp4"},
			},
			expected: true,
		},
		{
			name: "Contains only images",
			enclosures: EnclosureList{
				Enclosure{MimeType: "image/jpeg"},
				Enclosure{MimeType: "image/png"},
			},
			expected: false,
		},
		{
			name: "Contains only documents",
			enclosures: EnclosureList{
				Enclosure{MimeType: "application/pdf"},
				Enclosure{MimeType: "text/plain"},
			},
			expected: false,
		},
		{
			name:       "Empty list",
			enclosures: EnclosureList{},
			expected:   false,
		},
		{
			name: "Single audio enclosure",
			enclosures: EnclosureList{
				Enclosure{MimeType: "audio/wav"},
			},
			expected: true,
		},
		{
			name: "Single video enclosure",
			enclosures: EnclosureList{
				Enclosure{MimeType: "video/webm"},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.enclosures.ContainsAudioOrVideo()
			if result != tc.expected {
				t.Errorf("ContainsAudioOrVideo() = %v, want %v", result, tc.expected)
			}
		})
	}
}
