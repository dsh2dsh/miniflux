package mux

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeMux(t *testing.T) {
	mux := New()
	require.NotNil(t, mux)

	var result []string
	makeHandleFunc := func(s string) func(http.ResponseWriter, *http.Request) {
		return func(http.ResponseWriter, *http.Request) {
			result = append(result, s)
		}
	}

	makeMiddleware := func(s string) MiddlewareFunc {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result = append(result, s)
				next.ServeHTTP(w, r)
			})
		}
	}

	mux.NameHandleFunc("/liveness", makeHandleFunc("liveness"), "liveness").
		PrefixGroup("/miniflux", func(mux *ServeMux) {
			mux.Use(makeMiddleware("gzip"), makeMiddleware("authHandlers")).
				Use(makeMiddleware("userSession")).
				HandleFunc("/unread", makeHandleFunc("unread")).
				NameHandleFunc("GET /robots.txt", makeHandleFunc("robots"), "robots").
				NameHandleFunc("/images/", makeHandleFunc("images"), "images")

			mux.HandleFunc(
				"/accounts/ClientLogin", makeHandleFunc("accounts/ClientLogin")).
				PrefixGroup("/reader/api/0").
				Use(makeMiddleware("greader/CORS")).
				NameHandleFunc("/token", makeHandleFunc("greader/token"), "Token")
		}).
		HandleFunc("POST /mark-all-as-read", makeHandleFunc("mark-all-as-read")).
		NameHandleFunc("/starred", makeHandleFunc("starred"), "starred").
		PrefixGroup("/v1").
		Use(makeMiddleware("v1/CORS")).
		Use(makeMiddleware("v1/requestUser")).
		HandleFunc("/users", makeHandleFunc("v1/users")).
		NameHandleFunc("/users/{userID}/mark-all-as-read",
			makeHandleFunc("v1/users/mark-all-as-read"), "v1mark-all-as-read")

	mux.NameHandleFunc("/healthcheck", makeHandleFunc("healthcheck"),
		"healthcheck")

	mux.Use(makeMiddleware("adminOnly")).
		HandleFunc("/foo", makeHandleFunc("foo"))

	mux.Group().Use(makeMiddleware("something")).
		HandleFunc("/bar", makeHandleFunc("bar"))

	tests := []struct {
		name     string
		method   string
		endpoint string
		expected []string
		assert   func(t *testing.T, mux *ServeMux)
	}{
		{
			name:     "liveness",
			endpoint: "/liveness",
			expected: []string{"liveness"},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/liveness", mux.NamedPath("liveness"))
			},
		},
		{
			name:     "unread",
			endpoint: "/miniflux/unread",
			expected: []string{"gzip", "authHandlers", "userSession", "unread"},
		},
		{
			name:     "robots",
			endpoint: "/miniflux/robots.txt",
			expected: []string{"gzip", "authHandlers", "userSession", "robots"},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/miniflux/robots.txt", mux.NamedPath("robots"))
			},
		},
		{
			name:     "mark-all-as-read",
			method:   http.MethodPost,
			endpoint: "/miniflux/mark-all-as-read",
			expected: []string{
				"gzip", "authHandlers", "userSession", "mark-all-as-read",
			},
		},
		{
			name:     "starred",
			endpoint: "/miniflux/starred",
			expected: []string{"gzip", "authHandlers", "userSession", "starred"},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/miniflux/starred", mux.NamedPath("starred"))
			},
		},
		{
			name:     "healthcheck",
			endpoint: "/healthcheck",
			expected: []string{"healthcheck"},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/healthcheck", mux.NamedPath("healthcheck"))
			},
		},
		{
			name:     "users",
			endpoint: "/miniflux/v1/users",
			expected: []string{
				"gzip", "authHandlers", "userSession",
				"v1/CORS", "v1/requestUser", "v1/users",
			},
		},
		{
			name:     "v1mark-all-as-read",
			endpoint: "/miniflux/v1/users/123/mark-all-as-read",
			expected: []string{
				"gzip", "authHandlers", "userSession",
				"v1/CORS", "v1/requestUser", "v1/users/mark-all-as-read",
			},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/miniflux/v1/users/123/mark-all-as-read",
					mux.NamedPath("v1mark-all-as-read", "userID", "123"))
			},
		},
		{
			name:     "ClientLogin",
			endpoint: "/miniflux/accounts/ClientLogin",
			expected: []string{
				"gzip", "authHandlers", "userSession", "accounts/ClientLogin",
			},
		},
		{
			name:     "token",
			endpoint: "/miniflux/reader/api/0/token",
			expected: []string{
				"gzip", "authHandlers", "userSession",
				"greader/CORS", "greader/token",
			},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/miniflux/reader/api/0/token", mux.NamedPath("Token"))
			},
		},
		{
			name:     "foo",
			endpoint: "/foo",
			expected: []string{"adminOnly", "foo"},
		},
		{
			name:     "bar",
			endpoint: "/bar",
			expected: []string{"adminOnly", "something", "bar"},
		},
		{
			name:     "images",
			endpoint: "/miniflux/images/",
			expected: []string{"gzip", "authHandlers", "userSession", "images"},
			assert: func(t *testing.T, mux *ServeMux) {
				assert.Equal(t, "/miniflux/images/", mux.NamedPath("images"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result = result[:0]
			method := tt.method
			if method == "" {
				method = http.MethodGet
			}
			r := httptest.NewRequest(method, tt.endpoint, nil)
			handler, pattern := mux.Handler(r)
			require.NotNil(t, handler)
			assert.NotEmpty(t, pattern)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, tt.expected, result)
			if tt.assert != nil {
				tt.assert(t, mux)
			}
		})
	}
}
