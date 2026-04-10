// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package response // import "miniflux.app/v2/internal/http/response"

import (
	"bytes"
	"errors"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klauspost/compress/gzhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseHasCommonHeaders(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}

	for header, expected := range headers {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Fatalf(`Unexpected header value, got %q instead of %q`, actual, expected)
		}
	}
}

func TestBuildResponseWithCustomStatusCode(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithStatus(http.StatusNotAcceptable).Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	expectedStatusCode := http.StatusNotAcceptable
	if resp.StatusCode != expectedStatusCode {
		t.Fatalf(`Unexpected status code, got %d instead of %d`, resp.StatusCode, expectedStatusCode)
	}
}

func TestBuildResponseWithCustomHeader(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithHeader("X-My-Header", "Value").Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	expected := "Value"
	actual := resp.Header.Get("X-My-Header")
	if actual != expected {
		t.Fatalf(`Unexpected header value, got %q instead of %q`, actual, expected)
	}
}

func TestBuildResponseWithAttachment(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithAttachment("my_file.pdf").Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	expected := "attachment; filename=my_file.pdf"
	actual := resp.Header.Get("Content-Disposition")
	if actual != expected {
		t.Fatalf(`Unexpected header value, got %q instead of %q`, actual, expected)
	}
}

func TestBuildResponseWithAttachmentEscapesFilename(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	w := httptest.NewRecorder()
	require.NotNil(t, w)

	const filename = `a";filename="malware.exe`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithAttachment(filename).Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()
	require.NotNil(t, resp)

	mediaType, params, err := mime.ParseMediaType(
		resp.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	require.NotEmpty(t, params)
	assert.Equal(t, "attachment", mediaType, "Unexpected media type")
	assert.Equal(t, filename, params["filename"], "Unexpected filename")
}

func TestBuildResponseWithInlineEscapesFilename(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	w := httptest.NewRecorder()
	require.NotNil(t, w)

	const filename = `a";filename="malware.exe`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithInline(filename).Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()
	require.NotNil(t, resp)

	mediaType, params, err := mime.ParseMediaType(
		resp.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	require.NotEmpty(t, params)
	assert.Equal(t, "inline", mediaType, "Unexpected media type")
	assert.Equal(t, filename, params["filename"], "Unexpected filename")
}

func TestFormatContentDisposition(t *testing.T) {
	tests := []struct {
		name            string
		dispositionType string
		filename        string
		expected        string
	}{
		{
			name:            "empty filename returns bare type",
			dispositionType: "inline",
			expected:        "inline",
		},
		{
			name:            "simple filename",
			dispositionType: "attachment",
			filename:        "photo.jpg",
			expected:        `attachment; filename=photo.jpg`,
		},
		{
			name:            "filename with double quote",
			dispositionType: "inline",
			filename:        `a";filename="malware.exe`,
			expected:        `inline; filename="a\";filename=\"malware.exe"`,
		},
		{
			name:            "filename with spaces",
			dispositionType: "attachment",
			filename:        "my file.txt",
			expected:        `attachment; filename="my file.txt"`,
		},
		{
			name:            "non-ASCII filename",
			dispositionType: "attachment",
			filename:        "café.png",
			expected:        `attachment; filename*=utf-8''caf%C3%A9.png`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected,
				formatContentDisposition(tt.dispositionType, tt.filename))
		})
	}
}

func TestBuildResponseWithError(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithBodyAsError(errors.New("Some error")).Write()
	})

	handler.ServeHTTP(w, r)

	expectedBody := `Some error`
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Fatalf(`Unexpected body, got %s instead of %s`, actualBody, expectedBody)
	}
}

func TestBuildResponseWithByteBody(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithBodyAsBytes([]byte("body")).Write()
	})

	handler.ServeHTTP(w, r)

	expectedBody := `body`
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Fatalf(`Unexpected body, got %s instead of %s`, actualBody, expectedBody)
	}
}

func TestBuildResponseWithCachingEnabled(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithCaching("etag", 1*time.Minute, func(b *Builder) {
			b.WithBodyAsString("cached body")
			b.Write()
		})
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	expectedStatusCode := http.StatusOK
	if resp.StatusCode != expectedStatusCode {
		t.Fatalf(`Unexpected status code, got %d instead of %d`, resp.StatusCode, expectedStatusCode)
	}

	expectedBody := `cached body`
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Fatalf(`Unexpected body, got %s instead of %s`, actualBody, expectedBody)
	}

	expectedHeader := "public, immutable"
	actualHeader := resp.Header.Get("Cache-Control")
	if actualHeader != expectedHeader {
		t.Fatalf(`Unexpected cache control header, got %q instead of %q`, actualHeader, expectedHeader)
	}

	if resp.Header.Get("Expires") == "" {
		t.Fatalf(`Expires header should not be empty`)
	}
}

func TestBuildResponseWithCachingAndEtag(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("If-None-Match", `"etag"`)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithCaching("etag", 1*time.Minute, func(b *Builder) {
			b.WithBodyAsString("cached body")
			b.Write()
		})
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusNotModified, resp.StatusCode,
		"Unexpected status code")

	expectedBody := ``
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Fatalf(`Unexpected body, got %s instead of %s`, actualBody, expectedBody)
	}

	expectedHeader := "public, immutable"
	actualHeader := resp.Header.Get("Cache-Control")
	if actualHeader != expectedHeader {
		t.Fatalf(`Unexpected cache control header, got %q instead of %q`, actualHeader, expectedHeader)
	}

	if resp.Header.Get("Expires") == "" {
		t.Fatalf(`Expires header should not be empty`)
	}
}

func TestBuildResponseWithCompression(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	w := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithBodyAsBytes([]byte("body")).Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()
	assert.Empty(t, resp.Header.Get(gzhttp.HeaderNoCompression))
}

func TestBuildResponseWithCompressionDisabled(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	w := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithBodyAsBytes([]byte("body")).WithoutCompression().Write()
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()
	assert.NotEmpty(t, resp.Header.Get(gzhttp.HeaderNoCompression))
}

func TestBuildResponseWithReaderBody(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	require.NotNil(t, r)

	w := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		New(w, r).WithBodyAsReader(bytes.NewBufferString("body")).Write()
	})

	handler.ServeHTTP(w, r)
	assert.Equal(t, "body", w.Body.String())
}
