// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ui // import "miniflux.app/v2/internal/ui"

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/reader/rewrite"
)

func (h *handler) mediaProxy(w http.ResponseWriter, r *http.Request) {
	// If we receive a "If-None-Match" header, we assume the media is already
	// stored in browser cache.
	if r.Header.Get("If-None-Match") != "" {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	encodedDigest := request.RouteStringParam(r, "encodedDigest")
	encodedURL := request.RouteStringParam(r, "encodedURL")
	if encodedURL == "" {
		html.BadRequest(w, r, errors.New("no URL provided"))
		return
	}

	decodedDigest, err := base64.URLEncoding.DecodeString(encodedDigest)
	if err != nil {
		html.BadRequest(w, r, errors.New("unable to decode this digest"))
		return
	}

	decodedURL, err := base64.URLEncoding.DecodeString(encodedURL)
	if err != nil {
		html.BadRequest(w, r, errors.New("unable to decode this URL"))
		return
	}

	mac := hmac.New(sha256.New, config.Opts.MediaProxyPrivateKey())
	mac.Write(decodedURL)
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(decodedDigest, expectedMAC) {
		html.Forbidden(w, r)
		return
	}

	parsedMediaURL, err := url.Parse(string(decodedURL))
	if err != nil {
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	if parsedMediaURL.Scheme != "http" && parsedMediaURL.Scheme != "https" {
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	if parsedMediaURL.Host == "" {
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	if !parsedMediaURL.IsAbs() {
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	mediaURL := string(decodedURL)
	slog.Debug("MediaProxy: Fetching remote resource",
		slog.String("media_url", mediaURL),
	)

	req, err := http.NewRequest(http.MethodGet, mediaURL, nil)
	if err != nil {
		html.ServerError(w, r, err)
		return
	}

	req.Header.Set("Connection", "close")

	if referer := rewrite.GetRefererForURL(mediaURL); referer != "" {
		req.Header.Set("Referer", referer)
	}

	forwardedRequestHeader := []string{
		"Range", "Accept", "Accept-Encoding", "User-Agent",
	}
	for _, requestHeaderName := range forwardedRequestHeader {
		if r.Header.Get(requestHeaderName) != "" {
			req.Header.Set(requestHeaderName, r.Header.Get(requestHeaderName))
		}
	}

	transport := &http.Transport{
		IdleConnTimeout: time.Duration(config.Opts.MediaProxyHTTPClientTimeout()) * time.Second,
	}
	clt := &http.Client{
		Transport: gzhttp.Transport(transport),
		Timeout:   time.Duration(config.Opts.MediaProxyHTTPClientTimeout()) * time.Second,
	}

	resp, err := clt.Do(req)
	if err != nil {
		slog.Error("MediaProxy: Unable to initialize HTTP client",
			slog.String("media_url", mediaURL),
			slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		slog.Warn(
			"MediaProxy: "+http.StatusText(http.StatusRequestedRangeNotSatisfiable),
			slog.String("media_url", mediaURL),
			slog.Int("status_code", resp.StatusCode))
		html.RequestedRangeNotSatisfiable(w, r, resp.Header.Get("Content-Range"))
		return
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		slog.Warn("MediaProxy: Unexpected response status code",
			slog.String("media_url", mediaURL),
			slog.Int("status_code", resp.StatusCode))

		// Forward the status code from the origin.
		http.Error(w, fmt.Sprintf("Origin status code is %d", resp.StatusCode),
			resp.StatusCode)
		return
	}

	etag := crypto.HashFromBytes(decodedURL)

	response.New(w, r).WithCaching(etag, 72*time.Hour, func(b *response.Builder) {
		b.WithStatus(resp.StatusCode)
		b.WithHeader("Content-Security-Policy",
			response.ContentSecurityPolicyForUntrustedContent)
		b.WithHeader("Content-Type", resp.Header.Get("Content-Type"))

		if filename := path.Base(parsedMediaURL.Path); filename != "" {
			b.WithHeader("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		}

		forwardedResponseHeader := []string{
			"Content-Encoding", "Content-Type", "Content-Length", "Accept-Ranges",
			"Content-Range",
		}
		for _, responseHeaderName := range forwardedResponseHeader {
			if resp.Header.Get(responseHeaderName) != "" {
				b.WithHeader(responseHeaderName, resp.Header.Get(responseHeaderName))
			}
		}
		b.WithBody(resp.Body)
		b.WithoutCompression()
		b.Write()
	})
}
