package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_ok(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return map[string]string{"key": "value"}, nil
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"key":"value"}`, w.Body.String(), "Unexpected body")
}

func TestJSON_created(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := CreatedJSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return map[string]string{"key": "value"}, nil
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"key":"value"}`, w.Body.String(), "Unexpected body")
}

func TestJSON_error(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	const errorString = "some error"
	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return nil, errors.New(errorString)
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"`+errorString+`"}`, w.Body.String(),
		"Unexpected body")
}

func TestJSON_badRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	const errorString = "some error"
	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return nil, WrapBadRequest(errors.New(errorString))
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"`+errorString+`"}`, w.Body.String(),
		"Unexpected body")
}

func TestJSON_unauthorized(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return nil, ErrUnauthorized
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"Unauthorized"}`, w.Body.String(),
		"Unexpected body")
}

func TestJSON_forbidden(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return nil, ErrForbidden
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"Forbidden"}`, w.Body.String(),
		"Unexpected body")
}

func TestJSON_notFound(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
			return nil, ErrNotFound
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"Not Found"}`, w.Body.String(),
		"Unexpected body")
}

func TestJSON_invalid(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := JSON(
		func(w http.ResponseWriter, r *http.Request) (chan int, error) {
			return make(chan int), nil
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"json: unsupported type: chan int"}`,
		w.Body.String(), "Unexpected body")
}

func TestNoContentJSON_ok(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := NoContentJSON(
		func(w http.ResponseWriter, r *http.Request) error { return nil })

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode,
		"Unexpected status code")
}

func TestNoContentJSON_error(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	const errorString = "oops, something wrong"
	handler := NoContentJSON(
		func(w http.ResponseWriter, r *http.Request) error {
			return errors.New(errorString)
		})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
		"Unexpected status code")
	//nolint:testifylint // not a JSON
	assert.Equal(t, resp.Header.Get(contentType), applicationJSON,
		"Unexpected content type")
	assert.JSONEq(t, `{"error_message":"`+errorString+`"}`, w.Body.String(),
		"Unexpected body")
}
