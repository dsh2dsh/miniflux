// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miniflux.app/v2/internal/config"
)

func TestNewRequestBuilder(t *testing.T) {
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	require.NotNil(t, builder)
	assert.Equal(t, config.HTTPClientTimeout(), builder.clientTimeout)
	assert.NotNil(t, builder.headers)
}

func TestRequestBuilder_WithHeader(t *testing.T) {
	require.NoError(t, config.Load(""))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Custom-Header") != "custom-value" {
			t.Errorf("Expected Custom-Header to be 'custom-value', got '%s'", r.Header.Get("Custom-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	builder := NewRequestBuilder()
	resp, err := builder.WithHeader("Custom-Header", "custom-value").Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Close()
}

func TestRequestBuilder_WithETag(t *testing.T) {
	require.NoError(t, config.Load(""))

	tests := []struct {
		name     string
		etag     string
		expected string
	}{
		{"with etag", "test-etag", "test-etag"},
		{"empty etag", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-None-Match") != tt.expected {
					t.Errorf("Expected If-None-Match to be '%s', got '%s'", tt.expected, r.Header.Get("If-None-Match"))
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { server.Close() })

			builder := NewRequestBuilder()
			resp, err := builder.WithETag(tt.etag).Request(server.URL)
			require.NoError(t, err)
			require.NotNil(t, resp)
			t.Cleanup(func() { resp.Close() })
		})
	}
}

func TestRequestBuilder_WithLastModified(t *testing.T) {
	require.NoError(t, config.Load(""))

	tests := []struct {
		name         string
		lastModified string
		expected     string
	}{
		{"with last modified", "Mon, 02 Jan 2006 15:04:05 GMT", "Mon, 02 Jan 2006 15:04:05 GMT"},
		{"empty last modified", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-Modified-Since") != tt.expected {
					t.Errorf("Expected If-Modified-Since to be '%s', got '%s'", tt.expected, r.Header.Get("If-Modified-Since"))
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { server.Close() })

			builder := NewRequestBuilder()
			resp, err := builder.WithLastModified(tt.lastModified).Request(server.URL)
			require.NoError(t, err)
			require.NotNil(t, resp)
			t.Cleanup(func() { resp.Close() })
		})
	}
}

func TestRequestBuilder_WithUserAgent(t *testing.T) {
	require.NoError(t, config.Load(""))

	tests := []struct {
		name           string
		userAgent      string
		defaultAgent   string
		expectedHeader string
	}{
		{"custom user agent", "CustomAgent/1.0", "DefaultAgent/1.0", "CustomAgent/1.0"},
		{"default user agent", "", "DefaultAgent/1.0", "DefaultAgent/1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("User-Agent") != tt.expectedHeader {
					t.Errorf("Expected User-Agent to be '%s', got '%s'", tt.expectedHeader, r.Header.Get("User-Agent"))
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { server.Close() })

			builder := NewRequestBuilder()
			resp, err := builder.WithUserAgent(tt.userAgent, tt.defaultAgent).Request(server.URL)
			require.NoError(t, err)
			require.NotNil(t, resp)
			t.Cleanup(func() { resp.Close() })
		})
	}
}

func TestRequestBuilder_WithCookie(t *testing.T) {
	require.NoError(t, config.Load(""))

	tests := []struct {
		name     string
		cookie   string
		expected string
	}{
		{"with cookie", "session=abc123; lang=en", "session=abc123; lang=en"},
		{"empty cookie", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Cookie") != tt.expected {
					t.Errorf("Expected Cookie to be '%s', got '%s'", tt.expected, r.Header.Get("Cookie"))
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { server.Close() })

			builder := NewRequestBuilder()
			resp, err := builder.WithCookie(tt.cookie).Request(server.URL)
			require.NoError(t, err)
			require.NotNil(t, resp)
			t.Cleanup(func() { resp.Close() })
		})
	}
}

func TestRequestBuilder_WithUsernameAndPassword(t *testing.T) {
	require.NoError(t, config.Load(""))

	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{"with credentials", "test", "password", "Basic dGVzdDpwYXNzd29yZA=="}, // base64 of "test:password"
		{"empty username", "", "password", ""},
		{"empty password", "test", "", ""},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != tt.expected {
					t.Errorf("Expected Authorization to be '%s', got '%s'", tt.expected, r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { server.Close() })

			builder := NewRequestBuilder()
			resp, err := builder.WithUsernameAndPassword(tt.username, tt.password).Request(server.URL)
			require.NoError(t, err)
			require.NotNil(t, resp)
			t.Cleanup(func() { resp.Close() })
		})
	}
}

func TestRequestBuilder_DefaultAcceptHeader(t *testing.T) {
	require.NoError(t, config.Load(""))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != defaultAcceptHeader {
			t.Errorf("Expected Accept to be '%s', got '%s'", defaultAcceptHeader, r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	builder := NewRequestBuilder()
	resp, err := builder.Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Close()
}

func TestRequestBuilder_CustomAcceptHeaderNotOverridden(t *testing.T) {
	require.NoError(t, config.Load(""))

	customAccept := "application/json"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != customAccept {
			t.Errorf("Expected Accept to be '%s', got '%s'", customAccept, r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	builder := NewRequestBuilder()
	resp, err := builder.WithHeader("Accept", customAccept).Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Close()
}

func TestRequestBuilder_WithoutRedirects(t *testing.T) {
	require.NoError(t, config.Load(""))

	// Create a redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer redirectServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectServer.URL, http.StatusFound)
	}))
	defer server.Close()

	builder := NewRequestBuilder()
	resp, err := builder.WithoutRedirects().Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Close()

	if resp.StatusCode() != http.StatusFound {
		t.Errorf("Expected status code %d, got %d", http.StatusFound, resp.StatusCode())
	}
}

func TestRequestBuilder_DisableHTTP2(t *testing.T) {
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	builder = builder.DisableHTTP2(true)
	assert.True(t, builder.disableHTTP2)
}

func TestRequestBuilder_IgnoreTLSErrors(t *testing.T) {
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	builder = builder.IgnoreTLSErrors(true)
	assert.True(t, builder.ignoreTLSErrors)
}

func TestRequestBuilder_WithCustomApplicationProxyURL(t *testing.T) {
	const proxyURL = "http://proxy.example.com:8080"
	t.Setenv("HTTP_CLIENT_PROXY", proxyURL)
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	assert.Equal(t, proxyURL, builder.clientProxyURL.String())
}

func TestRequestBuilder_UseCustomApplicationProxyURL(t *testing.T) {
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	builder = builder.UseCustomApplicationProxyURL(true)
	assert.True(t, builder.useClientProxy)
}

func TestRequestBuilder_WithCustomFeedProxyURL(t *testing.T) {
	require.NoError(t, config.Load(""))

	proxyURL := "http://feed-proxy.example.com:8080"
	builder := NewRequestBuilder()
	builder = builder.WithCustomFeedProxyURL(proxyURL)
	assert.Equal(t, proxyURL, builder.feedProxyURL)
}

func TestRequestBuilder_ChainedMethods(t *testing.T) {
	require.NoError(t, config.Load(""))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check multiple headers
		if r.Header.Get("User-Agent") != "TestAgent/1.0" {
			t.Errorf("Expected User-Agent to be 'TestAgent/1.0', got '%s'", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Cookie") != "test=value" {
			t.Errorf("Expected Cookie to be 'test=value', got '%s'", r.Header.Get("Cookie"))
		}
		if r.Header.Get("If-None-Match") != "etag123" {
			t.Errorf("Expected If-None-Match to be 'etag123', got '%s'", r.Header.Get("If-None-Match"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	builder := NewRequestBuilder()
	resp, err := builder.
		WithUserAgent("TestAgent/1.0", "DefaultAgent/1.0").
		WithCookie("test=value").
		WithETag("etag123").
		Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Close()
}

func TestRequestBuilder_InvalidURL(t *testing.T) {
	require.NoError(t, config.Load(""))

	builder := NewRequestBuilder()
	_, err := builder.Request(":|invalid-url")
	t.Log(err)
	require.Error(t, err)
}

func TestDenyDialToPrivate(t *testing.T) {
	tests := []struct {
		name    string
		address string
		allow   bool
	}{
		{
			name:    "private IPv4",
			address: "192.168.1.10:8000",
		},
		{
			name:    "loopback IPv4",
			address: "127.0.0.1:8000",
		},
		{
			name:    "link-local IPv4",
			address: "169.254.42.1:8000",
		},
		{
			name:    "multicast IPv4",
			address: "224.0.0.1:8000",
		},
		{
			name:    "unspecified IPv6",
			address: "[::]:8000",
		},
		{
			name:    "loopback IPv6",
			address: "[::1]:8000",
		},
		{
			name:    "multicast IPv6",
			address: "[ff02::1]:8000",
		},
		{
			name:    "public IPv4",
			address: "93.184.216.34:8000",
			allow:   true,
		},
		{
			name:    "public IPv6",
			address: "[2001:4860:4860::8888]:8000",
			allow:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.allow {
				require.NoError(t, denyDialToPrivate("", tt.address, nil))
				return
			}
			require.ErrorIs(t, denyDialToPrivate("", tt.address, nil),
				errDialToPrivate)
		})
	}
}

func TestRequestBuilder_WithDenyPrivateNets(t *testing.T) {
	if testing.Verbose() {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	os.Clearenv()
	require.NoError(t, config.Load(""))

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	t.Cleanup(server.Close)

	rb := NewRequestBuilder()
	resp, err := rb.WithDenyPrivateNets(true).Request(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.ErrorIs(t, resp.Err(), errDialToPrivate)

	resp, err = rb.WithDenyPrivateNets(false).Request(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	resp.Close()
}
