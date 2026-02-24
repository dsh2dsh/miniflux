// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsModified(t *testing.T) {
	cachedEtag := "abc123"
	cachedLastModified := "Wed, 21 Oct 2015 07:28:00 GMT"

	testCases := map[string]struct {
		Status       int
		LastModified string
		ETag         string
		IsModified   bool
	}{
		"Unmodified 304": {
			Status:       304,
			LastModified: cachedLastModified,
			ETag:         cachedEtag,
			IsModified:   false,
		},
		"Unmodified 200": {
			Status:       200,
			LastModified: cachedLastModified,
			ETag:         cachedEtag,
			IsModified:   false,
		},
		// ETag takes precedence per RFC9110 8.8.1.
		"Last-Modified changed only": {
			Status:       200,
			LastModified: "Thu, 22 Oct 2015 07:28:00 GMT",
			ETag:         cachedEtag,
			IsModified:   false,
		},
		"ETag changed only": {
			Status:       200,
			LastModified: cachedLastModified,
			ETag:         "xyz789",
			IsModified:   true,
		},
		"ETag and Last-Modified changed": {
			Status:       200,
			LastModified: "Thu, 22 Oct 2015 07:28:00 GMT",
			ETag:         "xyz789",
			IsModified:   true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(tt *testing.T) {
			header := http.Header{}
			header.Add("Last-Modified", tc.LastModified)
			header.Add("ETag", tc.ETag)
			rh := ResponseHandler{
				httpResponse: &http.Response{
					StatusCode: tc.Status,
					Header:     header,
				},
			}
			if tc.IsModified != rh.IsModified(cachedEtag, cachedLastModified) {
				tt.Error(name)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	tests := [...]struct {
		name        string
		retryHeader string
		wantDelay   time.Duration
	}{
		{
			name:      "empty header",
			wantDelay: 0,
		},
		{
			name:        "garbage header",
			retryHeader: "foobar",
		},
		{
			name:        "integer value",
			retryHeader: "42",
			wantDelay:   42 * time.Second,
		},
		{
			name:        "negative value",
			retryHeader: "-42",
			wantDelay:   0,
		},
		{
			name:        "HTTP-date",
			retryHeader: time.Now().Add(42 * time.Second).Format(time.RFC1123),
			wantDelay:   41 * time.Second,
		},
		{
			name:        "past HTTP-date",
			retryHeader: time.Now().Add(-42 * time.Second).Format(time.RFC1123),
			wantDelay:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("Retry-After:", tt.retryHeader)
			header := http.Header{}
			header.Add("Retry-After", tt.retryHeader)
			resp := ResponseHandler{httpResponse: &http.Response{Header: header}}
			assert.Equal(t, tt.wantDelay,
				resp.parseRetryDelay().Truncate(time.Second))
		})
	}
}

func TestExpiresInMinutes(t *testing.T) {
	testCases := map[string]struct {
		ExpiresHeader   string
		ExpectedMinutes int
	}{
		"Empty header": {
			ExpiresHeader:   "",
			ExpectedMinutes: 0,
		},
		"Valid Expires header": {
			ExpiresHeader:   time.Now().Add(10 * time.Minute).Format(time.RFC1123),
			ExpectedMinutes: 10,
		},
		"Invalid Expires header": {
			ExpiresHeader:   "invalid-date",
			ExpectedMinutes: 0,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(tt *testing.T) {
			header := http.Header{}
			header.Add("Expires", tc.ExpiresHeader)
			rh := ResponseHandler{
				httpResponse: &http.Response{
					Header: header,
				},
			}
			if tc.ExpectedMinutes != rh.ExpiresInMinutes() {
				t.Errorf("Expected %d, got %d for scenario %q", tc.ExpectedMinutes, rh.ExpiresInMinutes(), name)
			}
		})
	}
}

func TestCacheControlMaxAgeInMinutes(t *testing.T) {
	testCases := map[string]struct {
		CacheControlHeader string
		ExpectedMinutes    int
	}{
		"Empty header": {
			CacheControlHeader: "",
			ExpectedMinutes:    0,
		},
		"Valid max-age": {
			CacheControlHeader: "max-age=600",
			ExpectedMinutes:    10,
		},
		"Invalid max-age": {
			CacheControlHeader: "max-age=invalid",
			ExpectedMinutes:    0,
		},
		"Multiple directives": {
			CacheControlHeader: "no-cache, max-age=300",
			ExpectedMinutes:    5,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(tt *testing.T) {
			header := http.Header{}
			header.Add("Cache-Control", tc.CacheControlHeader)
			rh := ResponseHandler{
				httpResponse: &http.Response{
					Header: header,
				},
			}
			if tc.ExpectedMinutes != rh.CacheControlMaxAgeInMinutes() {
				t.Errorf("Expected %d, got %d for scenario %q", tc.ExpectedMinutes, rh.CacheControlMaxAgeInMinutes(), name)
			}
		})
	}
}
