// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package json // import "miniflux.app/v2/internal/http/response/json"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/logging"
)

const (
	contentType       = "Content-Type"
	contentTypeHeader = "application/json"
)

// OK creates a new JSON response with a 200 status code.
func OK(w http.ResponseWriter, r *http.Request, body any) {
	responseBody, err := json.Marshal(body)
	if err != nil {
		ServerError(w, r, err)
		return
	}

	response.New(w, r).
		WithHeader(contentType, contentTypeHeader).
		WithBody(responseBody).
		Write()
}

// Created sends a created response to the client.
func Created(w http.ResponseWriter, r *http.Request, body any) {
	responseBody, err := json.Marshal(body)
	if err != nil {
		ServerError(w, r, err)
		return
	}

	response.New(w, r).
		WithStatus(http.StatusCreated).
		WithHeader(contentType, contentTypeHeader).
		WithBody(responseBody).
		Write()
}

// NoContent sends a no content response to the client.
func NoContent(w http.ResponseWriter, r *http.Request) {
	response.New(w, r).
		WithStatus(http.StatusNoContent).
		WithHeader(contentType, contentTypeHeader).
		Write()
}

func Accepted(w http.ResponseWriter, r *http.Request) {
	response.New(w, r).
		WithStatus(http.StatusAccepted).
		WithHeader(contentType, contentTypeHeader).
		Write()
}

// ServerError sends an internal error to the client.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	log := logging.FromContext(r.Context()).With(
		slog.Any("error", err),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent())))

	clientClosed := errors.Is(err, context.Canceled) &&
		errors.Is(r.Context().Err(), context.Canceled)
	if clientClosed {
		statusCode := 499
		log.Debug("client closed request",
			slog.Group("response", slog.Int("status_code", statusCode)))
		http.Error(w, err.Error(), statusCode)
		return
	}

	statusCode := http.StatusInternalServerError
	log.Error(http.StatusText(statusCode),
		slog.Group("response",
			slog.Int("status_code", statusCode)))

	body, ok := generateJSONError(w, r, err)
	if !ok {
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, contentTypeHeader).
		WithBody(body).
		Write()
}

// BadRequest sends a bad request error to the client.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusBadRequest
	logStatusCode(r, statusCode, err)

	body, ok := generateJSONError(w, r, err)
	if !ok {
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, contentTypeHeader).
		WithBody(body).
		Write()
}

func logStatusCode(r *http.Request, statusCode int, err error) {
	log := logging.FromContext(r.Context())
	if err != nil {
		log = log.With(slog.Any("error", err))
	}
	log.Warn(http.StatusText(statusCode),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent())),
		slog.Group("response",
			slog.Int("status_code", statusCode)))
}

// Unauthorized sends a not authorized error to the client.
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusUnauthorized
	logStatusCode(r, statusCode, nil)

	body, ok := generateJSONError(w, r, errors.New("access unauthorized"))
	if !ok {
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, contentTypeHeader).
		WithBody(body).
		Write()
}

// Forbidden sends a forbidden error to the client.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusForbidden
	logStatusCode(r, statusCode, nil)

	body, ok := generateJSONError(w, r, errors.New("access forbidden"))
	if !ok {
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, contentTypeHeader).
		WithBody(body).
		Write()
}

// NotFound sends a page not found error to the client.
func NotFound(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusNotFound
	logStatusCode(r, statusCode, nil)

	body, ok := generateJSONError(w, r, errors.New("resource not found"))
	if !ok {
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader(contentType, contentTypeHeader).
		WithBody(body).
		Write()
}

func generateJSONError(w http.ResponseWriter, r *http.Request, err error,
) ([]byte, bool) {
	type errorMsg struct {
		ErrorMessage string `json:"error_message"`
	}

	body, err := json.Marshal(errorMsg{ErrorMessage: err.Error()})
	if err != nil {
		logging.FromContext(r.Context()).Error("Unable to generate JSON error",
			slog.Any("error", fmt.Errorf(
				"http/response/json: failed marshal error message: %w", err)))
		statusCode := http.StatusInternalServerError
		http.Error(w, http.StatusText(statusCode), statusCode)
		return nil, false
	}
	return body, true
}
