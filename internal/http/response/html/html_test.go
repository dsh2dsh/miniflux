// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package html // import "miniflux.app/v2/internal/http/response/html"

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOKResponse(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		OK(w, r, []byte("Some HTML"))
	})

	handler.ServeHTTP(w, r)
	resp := w.Result()

	expectedStatusCode := http.StatusOK
	if resp.StatusCode != expectedStatusCode {
		t.Fatalf(`Unexpected status code, got %d instead of %d`, resp.StatusCode, expectedStatusCode)
	}

	expectedBody := `Some HTML`
	actualBody := w.Body.String()
	if actualBody != expectedBody {
		t.Fatalf(`Unexpected body, got %s instead of %s`, actualBody, expectedBody)
	}

	headers := map[string]string{
		"Content-Type":  "text/html; charset=utf-8",
		"Cache-Control": "no-cache, max-age=0, must-revalidate, no-store",
	}

	for header, expected := range headers {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Fatalf(`Unexpected header value, got %q instead of %q`, actual, expected)
		}
	}
}
