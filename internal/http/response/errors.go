package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strconv"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/logging"
)

const statusClientClosed = 499

var (
	ErrForbidden    = NewError(http.StatusForbidden)
	ErrNotFound     = NewError(http.StatusNotFound)
	ErrUnauthorized = NewError(http.StatusUnauthorized)
)

type Error struct {
	Status     string
	StatusCode int

	err error
}

var (
	_ error          = (*Error)(nil)
	_ json.Marshaler = (*Error)(nil)
)

func NewError(statusCode int) *Error {
	return &Error{StatusCode: statusCode}
}

func WrapBadRequest(err error) *Error {
	return WrapError(err, http.StatusBadRequest)
}

func WrapError(err error, statusCode int) *Error {
	if e, ok := errors.AsType[*Error](err); ok {
		return e
	}
	return &Error{StatusCode: statusCode, err: err}
}

func WrapServerError(err error) *Error {
	return WrapError(err, http.StatusInternalServerError)
}

func (self *Error) WithStatus(status string) *Error {
	self.Status = status
	return self
}

func (self *Error) Error() string {
	switch {
	case self.err != nil:
		return self.err.Error()
	case self.Status != "":
		return self.Status
	}
	return http.StatusText(self.StatusCode)
}

func (self *Error) Unwrap() error {
	if self.err == nil {
		return nil
	}
	return self.err
}

func (self *Error) MarshalJSON() ([]byte, error) {
	e := struct {
		ErrorMessage string `json:"error_message"`
	}{
		ErrorMessage: self.Error(),
	}

	b, err := json.Marshal(&e)
	if err != nil {
		return nil, fmt.Errorf("http/response: marshal error: %w", err)
	}
	return b, nil
}

func (self *Error) String() string {
	if self.err == nil {
		return self.statusText()
	}
	return self.statusText() + ": " + self.err.Error()
}

func (self *Error) statusText() string {
	if self.Status != "" {
		return strconv.Itoa(self.StatusCode) + " " + self.Status
	}
	return strconv.Itoa(self.StatusCode) + " " + http.StatusText(self.StatusCode)
}

func (self *Error) Log(r *http.Request) {
	log := logging.FromContext(r.Context()).With(
		slog.String("client_ip", request.ClientIP(r)),
		slog.GroupAttrs("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent())))

	if self.err == nil {
		log.Error(self.statusText(),
			slog.GroupAttrs("response", slog.Int("status_code", self.StatusCode)))
		return
	}

	if self.ClientClosed(r) {
		log.Debug("client closed request",
			slog.GroupAttrs("response", slog.Int("status_code", statusClientClosed)),
			slog.Any("error", self.err))
		return
	}

	log.Error(self.statusText(),
		slog.GroupAttrs("response", slog.Int("status_code", self.StatusCode)),
		slog.Any("error", self.err))
}

func (self *Error) ClientClosed(r *http.Request) bool {
	return errors.Is(self.err, context.Canceled) &&
		errors.Is(r.Context().Err(), context.Canceled)
}

func (self *Error) Serve(w http.ResponseWriter, r *http.Request,
	opts ...Option,
) {
	self.Log(r)
	if self.ClientClosed(r) {
		http.Error(w, self.Error(), statusClientClosed)
		return
	}
	self.Response(w, r, opts...).Write()
}

func (self *Error) Response(w http.ResponseWriter, r *http.Request,
	opts ...Option,
) *Builder {
	b := New(w, r, opts...).
		WithStatus(self.StatusCode).
		WithHeader(contentSecPol, ContentSecurityPolicyForUntrustedContent).
		WithHeader(contentType, textPlain).
		WithHeader(cacheControl, cacheNoCache).
		WithBodyAsString(html.EscapeString(self.String()))
	return b
}

func (self *Error) ServeJSON(w http.ResponseWriter, r *http.Request,
	opts ...Option,
) {
	self.Log(r)
	if self.ClientClosed(r) {
		http.Error(w, self.Error(), statusClientClosed)
		return
	}

	b, err := self.MarshalJSON()
	if err != nil {
		logging.FromContext(r.Context()).Error(
			"Unable generate JSON error response",
			slog.Any("error", err))
		self.Response(w, r, opts...).Write()
		return
	}

	New(w, r, opts...).
		WithStatus(self.StatusCode).
		WithHeader(contentType, applicationJSON).
		WithBodyAsBytes(b).
		Write()
}
