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

var translatedStatusCodes = map[int]string{
	http.StatusUnauthorized:        "error.http_not_authorized",
	http.StatusForbidden:           "error.http_forbidden",
	http.StatusNotFound:            "error.http_resource_not_found",
	http.StatusGone:                "error.http_resource_not_found",
	http.StatusInternalServerError: "error.http_internal_server_error",
	http.StatusBadGateway:          "error.http_bad_gateway",
	http.StatusServiceUnavailable:  "error.http_service_unavailable",
	http.StatusGatewayTimeout:      "error.http_gateway_timeout",
}

func (r *ResponseHandler) LocalizedError() *locale.LocalizedErrorWrapper {
	if r.Err() != nil {
		return localizeClientErr(r.Err())
	}
	return r.localizeStatusCode()
}

func localizeClientErr(err error) *locale.LocalizedErrorWrapper {
	const msgFmt = "reader/fetcher: http client error: %w"
	switch {
	case sslError(err):
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf(msgFmt, err), "error.tls_error", err)
	case networkError(err):
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf(msgFmt, err), "error.network_operation", err)
	case os.IsTimeout(err):
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf(msgFmt, err), "error.network_timeout", err)
	case errors.Is(err, io.EOF):
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf(msgFmt, err), "error.http_empty_response")
	}
	return locale.NewLocalizedErrorWrapper(
		fmt.Errorf(msgFmt, err), "error.http_client_error", err)
}

func networkError(err error) bool {
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

func sslError(err error) bool {
	var certErr *x509.UnknownAuthorityError
	if errors.As(err, &certErr) {
		return true
	}

	var hostErr *x509.HostnameError
	if errors.As(err, &hostErr) {
		return true
	}

	var algErr *x509.InsecureAlgorithmError
	return errors.As(err, &algErr)
}

func (r *ResponseHandler) localizeStatusCode() *locale.LocalizedErrorWrapper {
	statusCode := r.StatusCode()
	if statusCode < 400 {
		if statusCode != http.StatusNotModified {
			// Content-Length = -1 when no Content-Length header is sent.
			if r.httpResponse.ContentLength == 0 {
				return locale.NewLocalizedErrorWrapper(
					errors.New("reader/fetcher: empty response body"),
					"error.http_empty_response_body")
			}
		}
		return nil
	}

	statusText := r.bodyStatusText()
	if key, ok := translatedStatusCodes[statusCode]; ok {
		return locale.NewLocalizedErrorWrapper(
			fmt.Errorf("reader/fetcher: unexpected response: %d %s",
				statusCode, statusText),
			key)
	}

	if statusCode == http.StatusTooManyRequests {
		err := fmt.Errorf("%w: %d %s",
			NewErrTooManyRequests(r.URL().Hostname(),
				time.Now().Add(r.parseRetryDelay())),
			statusCode, statusText)
		return locale.NewLocalizedErrorWrapper(err, "error.http_too_many_requests")
	}

	return locale.NewLocalizedErrorWrapper(
		fmt.Errorf("reader/fetcher: unexpected status code: %d %s",
			statusCode, statusText),
		"error.http_unexpected_status_code", statusCode)
}

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
	return fmt.Sprintf(
		"reader/fetcher: host %q rate limited, retry in %s",
		self.hostname, time.Until(self.RetryAfter()))
}

func (self *ErrTooManyRequests) RetryAfter() time.Time {
	return self.retryAfter
}

func (self *ErrTooManyRequests) Localized() *locale.LocalizedErrorWrapper {
	return locale.NewLocalizedErrorWrapper(self, "error.http_too_many_requests")
}
