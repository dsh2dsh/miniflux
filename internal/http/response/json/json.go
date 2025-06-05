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

const contentTypeHeader = `application/json`

// OK creates a new JSON response with a 200 status code.
func OK(w http.ResponseWriter, r *http.Request, body any) {
	responseBody, err := json.Marshal(body)
	if err != nil {
		ServerError(w, r, err)
		return
	}

	builder := response.New(w, r)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

// Created sends a created response to the client.
func Created(w http.ResponseWriter, r *http.Request, body any) {
	responseBody, err := json.Marshal(body)
	if err != nil {
		ServerError(w, r, err)
		return
	}

	builder := response.New(w, r)
	builder.WithStatus(http.StatusCreated)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

// NoContent sends a no content response to the client.
func NoContent(w http.ResponseWriter, r *http.Request) {
	builder := response.New(w, r)
	builder.WithStatus(http.StatusNoContent)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.Write()
}

func Accepted(w http.ResponseWriter, r *http.Request) {
	builder := response.New(w, r)
	builder.WithStatus(http.StatusAccepted)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.Write()
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

	body, jsonErr := generateJSONError(err)
	if jsonErr != nil {
		logging.FromContext(r.Context()).Error("Unable to generate JSON error",
			slog.Any("error", jsonErr))
		http.Error(w, http.StatusText(statusCode), statusCode)
		return
	}

	response.New(w, r).
		WithStatus(statusCode).
		WithHeader("Content-Type", contentTypeHeader).
		WithBody(body).
		Write()
}

// BadRequest sends a bad request error to the client.
func BadRequest(w http.ResponseWriter, r *http.Request, err error) {
	slog.Warn(http.StatusText(http.StatusBadRequest),
		slog.Any("error", err),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent()),
		),
		slog.Group("response",
			slog.Int("status_code", http.StatusBadRequest),
		),
	)

	responseBody, jsonErr := generateJSONError(err)
	if jsonErr != nil {
		slog.Error("Unable to generate JSON error", slog.Any("error", jsonErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	builder := response.New(w, r)
	builder.WithStatus(http.StatusBadRequest)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

// Unauthorized sends a not authorized error to the client.
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	slog.Warn(http.StatusText(http.StatusUnauthorized),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent()),
		),
		slog.Group("response",
			slog.Int("status_code", http.StatusUnauthorized),
		),
	)

	responseBody, jsonErr := generateJSONError(errors.New("access unauthorized"))
	if jsonErr != nil {
		slog.Error("Unable to generate JSON error", slog.Any("error", jsonErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	builder := response.New(w, r)
	builder.WithStatus(http.StatusUnauthorized)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

// Forbidden sends a forbidden error to the client.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	slog.Warn(http.StatusText(http.StatusForbidden),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent()),
		),
		slog.Group("response",
			slog.Int("status_code", http.StatusForbidden),
		),
	)

	responseBody, jsonErr := generateJSONError(errors.New("access forbidden"))
	if jsonErr != nil {
		slog.Error("Unable to generate JSON error", slog.Any("error", jsonErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	builder := response.New(w, r)
	builder.WithStatus(http.StatusForbidden)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

// NotFound sends a page not found error to the client.
func NotFound(w http.ResponseWriter, r *http.Request) {
	slog.Warn(http.StatusText(http.StatusNotFound),
		slog.String("client_ip", request.ClientIP(r)),
		slog.Group("request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("user_agent", r.UserAgent()),
		),
		slog.Group("response",
			slog.Int("status_code", http.StatusNotFound),
		),
	)

	responseBody, jsonErr := generateJSONError(errors.New("resource not found"))
	if jsonErr != nil {
		slog.Error("Unable to generate JSON error", slog.Any("error", jsonErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	builder := response.New(w, r)
	builder.WithStatus(http.StatusNotFound)
	builder.WithHeader("Content-Type", contentTypeHeader)
	builder.WithBody(responseBody)
	builder.Write()
}

func generateJSONError(err error) ([]byte, error) {
	type errorMsg struct {
		ErrorMessage string `json:"error_message"`
	}
	encodedBody, err := json.Marshal(errorMsg{ErrorMessage: err.Error()})
	if err != nil {
		return nil, fmt.Errorf(
			"http/response/json: failed marshal error message: %w", err)
	}
	return encodedBody, nil
}
