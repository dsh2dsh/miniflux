package fetcher

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"miniflux.app/v2/internal/locale"
)

type ErrTooManyRequests struct {
	hostname   string
	retryAfter time.Time
}

var _ error = (*ErrTooManyRequests)(nil)

func NewErrTooManyRequests(hostname string, retryAfter time.Time,
) *ErrTooManyRequests {
	return &ErrTooManyRequests{
		hostname:   hostname,
		retryAfter: retryAfter,
	}
}

func (self *ErrTooManyRequests) Error() string {
	return fmt.Sprintf("host %q rate limited: 429 %s",
		self.hostname, http.StatusText(http.StatusTooManyRequests))
}

func (self *ErrTooManyRequests) RetryAfter() time.Time {
	return self.retryAfter
}

func (self *ErrTooManyRequests) Localized() *locale.LocalizedErrorWrapper {
	return locale.NewLocalizedErrorWrapper(self, "error.http_too_many_requests")
}

func (r *ResponseHandler) LocalizedError() *locale.LocalizedErrorWrapper {
	if r.clientErr != nil {
		switch {
		case isSSLError(r.clientErr):
			return locale.NewLocalizedErrorWrapper(fmt.Errorf("fetcher: %w", r.clientErr), "error.tls_error", r.clientErr)
		case isNetworkError(r.clientErr):
			return locale.NewLocalizedErrorWrapper(fmt.Errorf("fetcher: %w", r.clientErr), "error.network_operation", r.clientErr)
		case os.IsTimeout(r.clientErr):
			return locale.NewLocalizedErrorWrapper(fmt.Errorf("fetcher: %w", r.clientErr), "error.network_timeout", r.clientErr)
		case errors.Is(r.clientErr, io.EOF):
			return locale.NewLocalizedErrorWrapper(fmt.Errorf("fetcher: %w", r.clientErr), "error.http_empty_response")
		default:
			return locale.NewLocalizedErrorWrapper(fmt.Errorf("fetcher: %w", r.clientErr), "error.http_client_error", r.clientErr)
		}
	}

	switch r.httpResponse.StatusCode {
	case http.StatusUnauthorized:
		return locale.NewLocalizedErrorWrapper(
			errors.New("fetcher: access unauthorized (401 status code)"),
			"error.http_not_authorized")
	case http.StatusForbidden:
		return locale.NewLocalizedErrorWrapper(
			errors.New("fetcher: access forbidden (403 status code)"),
			"error.http_forbidden")
	case http.StatusTooManyRequests:
		return locale.NewLocalizedErrorWrapper(
			errors.New("fetcher: too many requests (429 status code)"),
			"error.http_too_many_requests")
	case http.StatusNotFound, http.StatusGone:
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: resource not found (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_resource_not_found")
	case http.StatusInternalServerError:
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: remote server error (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_internal_server_error")
	case http.StatusBadGateway:
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: bad gateway (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_bad_gateway")
	case http.StatusServiceUnavailable:
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: service unavailable (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_service_unavailable")
	case http.StatusGatewayTimeout:
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: gateway timeout (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_gateway_timeout")
	}

	if r.httpResponse.StatusCode >= 400 {
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("fetcher: unexpected status code (%d status code)",
				r.httpResponse.StatusCode),
			"error.http_unexpected_status_code", r.httpResponse.StatusCode)
	}

	if r.httpResponse.StatusCode != http.StatusNotModified {
		// Content-Length = -1 when no Content-Length header is sent.
		if r.httpResponse.ContentLength == 0 {
			return locale.NewLocalizedErrorWrapper(
				errors.New("fetcher: empty response body"),
				"error.http_empty_response_body")
		}
	}
	return nil
}

func isNetworkError(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func isSSLError(err error) bool {
	var certErr x509.UnknownAuthorityError
	if errors.As(err, &certErr) {
		return true
	}

	var hostErr x509.HostnameError
	if errors.As(err, &hostErr) {
		return true
	}

	var algErr x509.InsecureAlgorithmError
	return errors.As(err, &algErr)
}
