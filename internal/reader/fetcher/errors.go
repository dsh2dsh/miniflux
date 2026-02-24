package fetcher

import (
	"bytes"
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
	"miniflux.app/v2/internal/reader/encoding"
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
	if _, ok := errors.AsType[*url.Error](err); ok {
		return true
	}

	if errors.Is(err, io.EOF) {
		return true
	}

	_, ok := errors.AsType[*net.OpError](err)
	return ok
}

func sslError(err error) bool {
	if _, ok := errors.AsType[*x509.UnknownAuthorityError](err); ok {
		return true
	}

	if _, ok := errors.AsType[*x509.HostnameError](err); ok {
		return true
	}

	_, ok := errors.AsType[*x509.InsecureAlgorithmError](err)
	return ok
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

	if key, ok := translatedStatusCodes[statusCode]; ok {
		return locale.NewLocalizedErrorWrapper(r.errResponse(), key)
	}

	if statusCode == http.StatusTooManyRequests {
		err := fmt.Errorf("%w: %w",
			NewErrTooManyRequests(r.URL().Hostname(),
				time.Now().Add(r.parseRetryDelay()), r.Header("Retry-After")),
			r.errResponse())
		return locale.NewLocalizedErrorWrapper(err, "error.http_too_many_requests")
	}

	return locale.NewLocalizedErrorWrapper(r.errResponse(),
		"error.http_unexpected_status_code", statusCode)
}

func (r *ResponseHandler) errResponse() *ErrResponse {
	errResp := &ErrResponse{
		StatusCode:  r.StatusCode(),
		ContentType: r.ContentType(),
	}
	if r.httpResponse.ContentLength == 0 {
		return errResp
	}

	body, err := encoding.NewCharsetReader(r.Body(), errResp.ContentType)
	if err != nil {
		return errResp
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, body); err != nil {
		return errResp
	}
	errResp.Body = b.Bytes()
	return errResp
}

type ErrTooManyRequests struct {
	hostname    string
	retryAfter  time.Time
	retryHeader string
}

var _ error = (*ErrTooManyRequests)(nil)

func NewErrTooManyRequests(hostname string, retryAfter time.Time,
	header string,
) *ErrTooManyRequests {
	return &ErrTooManyRequests{
		hostname:    hostname,
		retryAfter:  retryAfter,
		retryHeader: header,
	}
}

func (self *ErrTooManyRequests) Error() string {
	return fmt.Sprintf(
		"host %q rate limited, retry in %s, Retry-After=%q",
		self.hostname, self.Until(), self.retryHeader)
}

func (self *ErrTooManyRequests) Until() time.Duration {
	if time.Now().Before(self.retryAfter) {
		return time.Until(self.retryAfter)
	}
	return 0
}

func (self *ErrTooManyRequests) RetryAfter() time.Time {
	return self.retryAfter
}

func (self *ErrTooManyRequests) RetryHeader() string { return self.retryHeader }

func (self *ErrTooManyRequests) Localized() *locale.LocalizedErrorWrapper {
	return locale.NewLocalizedErrorWrapper(self, "error.http_too_many_requests")
}

type ErrResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

var _ error = (*ErrResponse)(nil)

func (self *ErrResponse) Error() string {
	return fmt.Sprintf("reader/fetcher: unexpected status: %d %s",
		self.StatusCode, http.StatusText(self.StatusCode))
}
