package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"miniflux.app/v2/internal/reader/fetcher"
)

// requestBuilder builds and executes HTTP requests with the builder pattern.
type requestBuilder struct {
	err      error
	endpoint string
	method   string
	body     io.Reader
	headers  http.Header
}

// NewRequestBuilder creates a new request builder for the given endpoint.
func NewRequestBuilder(endpoint string) *requestBuilder {
	return &requestBuilder{
		endpoint: endpoint,
		method:   http.MethodGet,
		headers:  make(http.Header),
	}
}

// WithMethod sets the HTTP method.
func (r *requestBuilder) WithMethod(method string) *requestBuilder {
	r.method = method
	return r
}

// WithHeader sets a header value.
func (r *requestBuilder) WithHeader(key, value string) *requestBuilder {
	r.headers.Set(key, value)
	return r
}

// WithJSON marshals payload as JSON, sets the body and Content-Type.
func (r *requestBuilder) WithJSON(payload any) *requestBuilder {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		r.err = fmt.Errorf("unable to encode request body: %w", err)
		return r
	}
	return r.WithJSONBody(requestBody)
}

// WithJSONBody sets an already-marshaled JSON body and the Content-Type.
// It is useful when the caller needs the encoded payload for another
// purpose (e.g. computing a signature) to avoid marshaling it twice.
func (r *requestBuilder) WithJSONBody(body []byte) *requestBuilder {
	r.body = bytes.NewReader(body)
	r.headers.Set("Content-Type", "application/json")
	return r
}

// Do builds and executes the request.
//
// Private networks are blocked unless explicitly allowed through the
// INTEGRATION_ALLOW_PRIVATE_NETWORKS option.
func (r *requestBuilder) Do(ctx context.Context) (*fetcher.ResponseHandler,
	error,
) {
	if r.err != nil {
		return nil, r.err
	}

	// The request is assembled lazily here rather than being stored as a
	// prebuilt *http.Request in the builder: http.NewRequest inspects the
	// body's concrete type (e.g. *bytes.Reader) to populate ContentLength and
	// GetBody. Constructing it only once the body is known yields a correct
	// Content-Length header and lets the client replay the body on redirects.
	req, err := http.NewRequestWithContext(ctx, r.method, r.endpoint, r.body)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	for key, values := range r.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return fetcher.Do(req, fetcher.WithIntegrationDefaults())
}
