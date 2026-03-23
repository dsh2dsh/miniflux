package response

import (
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadRequestResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	const errorString = "Some error with injected HTML <script>alert('XSS')</script>"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		BadRequest(w, r, errors.New(errorString))
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"Unexpected status code")
	assert.Equal(t, textPlain, resp.Header.Get(contentType),
		"Unexpected content type")
	assert.Equal(t,
		"400 "+http.StatusText(resp.StatusCode)+": "+html.EscapeString(errorString),
		w.Body.String(), "Unexpected body")
}

func TestForbiddenResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Forbidden(w, r)
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"Unexpected status code")
	assert.Equal(t, textPlain, resp.Header.Get(contentType),
		"Unexpected content type")
	assert.Equal(t, "403 Forbidden", w.Body.String(), "Unexpected body")
}

func TestNotFoundResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		NotFound(w, r)
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode,
		"Unexpected status code")
	assert.Equal(t, textPlain, resp.Header.Get(contentType),
		"Unexpected content type")
	assert.Equal(t, "404 Not Found", w.Body.String(), "Unexpected body")
}

func TestRedirectResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Redirect(w, r, "/path")
	})

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode, "Unexpected status code")
	assert.Equal(t, "/path", resp.Header.Get("Location"),
		"Unexpected redirect location")
}

func TestRequestedRangeNotSatisfiable(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	const contentRange = "bytes */12777"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequestedRangeNotSatisfiable(w, r, contentRange)
	})

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestedRangeNotSatisfiable, resp.StatusCode,
		"Unexpected status code")
	assert.Equal(t, contentRange, resp.Header.Get("Content-Range"),
		"Unexpected content range header")
}

func TestServerErrorResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	const errorString = "Some error with injected HTML <script>alert('XSS')</script>"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServerError(w, r, errors.New(errorString))
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
		"Unexpected status code")
	assert.Equal(t, textPlain, resp.Header.Get(contentType),
		"Unexpected content type")
	assert.Equal(t,
		"500 "+http.StatusText(resp.StatusCode)+": "+html.EscapeString(errorString),
		w.Body.String(), "Unexpected body")
}

func TestTextResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	const body = "Some plain text"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Text(w, r, body)
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
	assert.Equal(t, textPlain, resp.Header.Get(contentType),
		"Unexpected content type")
	assert.Equal(t, body, w.Body.String(), "Unexpected body")
}
