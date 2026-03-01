package mediaproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/http/request"
	"miniflux.app/v2/internal/http/response"
	"miniflux.app/v2/internal/http/response/html"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/reader/fetcher"
	"miniflux.app/v2/internal/reader/rewrite"
)

var proxyRequestHeaders = [...]string{
	"Accept",
	"Accept-Encoding",
	"Range",
	"User-Agent",
}

var proxyResponseHeaders = [...]string{
	"Accept-Ranges",
	"Content-Encoding",
	"Content-Length",
	"Content-Range",
	"Content-Type",
}

func Serve(w http.ResponseWriter, r *http.Request) {
	// If we receive a "If-None-Match" header, we assume the media is already
	// stored in browser cache.
	if r.Header.Get("If-None-Match") != "" {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	encodedURL := request.RouteStringParam(r, "encodedURL")
	if encodedURL == "" {
		html.BadRequest(w, r, errors.New("no URL provided"))
		return
	}

	encodedDigest := request.RouteStringParam(r, "encodedDigest")
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

	mac := hmac.New(sha256.New, config.MediaProxyPrivateKey())
	mac.Write(decodedURL)
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(decodedDigest, expectedMAC) {
		html.Forbidden(w, r)
		return
	}

	u, err := url.Parse(string(decodedURL))
	if err != nil {
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		fallthrough
	case u.Hostname() == "":
		fallthrough
	case !u.IsAbs():
		html.BadRequest(w, r, errors.New("invalid URL provided"))
		return
	}

	mediaURL := string(decodedURL)
	log := logging.FromContext(r.Context()).With(
		slog.String("media_url", mediaURL))
	log.Debug("MediaProxy: Fetching remote resource")

	rb := fetcher.NewRequestBuilder()

	if referer := rewrite.GetRefererForURL(mediaURL); referer != "" {
		rb.WithHeader("Referer", referer)
	}

	for _, name := range proxyRequestHeaders {
		if s := r.Header.Get(name); s != "" {
			rb.WithHeader(name, s)
		}
	}

	resp, err := rb.RequestWithContext(r.Context(), mediaURL)
	if err != nil {
		log.Error("MediaProxy: Unable to initialize HTTP client",
			slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}
	defer resp.Close()

	switch statusCode := resp.StatusCode(); statusCode {
	case http.StatusRequestedRangeNotSatisfiable:
		log.Warn("MediaProxy: "+http.StatusText(statusCode),
			slog.Int("status_code", statusCode))
		html.RequestedRangeNotSatisfiable(w, r, resp.Header("Content-Range"))
		return

	case http.StatusOK, http.StatusPartialContent:
	// everything is OK, do nothing

	default:
		log.Warn("MediaProxy: Unexpected response status code",
			slog.Int("status_code", statusCode))

		// Forward the status code from the origin.
		http.Error(w, "Origin status code is "+strconv.Itoa(statusCode), statusCode)
		return
	}

	etag := crypto.HashFromBytes(decodedURL)

	response.New(w, r).WithCaching(etag, 72*time.Hour, func(b *response.Builder) {
		b.WithStatus(resp.StatusCode())
		b.WithHeader("Content-Security-Policy",
			response.ContentSecurityPolicyForUntrustedContent)
		b.WithHeader("Content-Type", resp.Header("Content-Type"))

		if filename := path.Base(u.EscapedPath()); filename != "" {
			b.WithHeader("Content-Disposition", `inline; filename="`+filename+`"`)
		}

		for _, name := range proxyResponseHeaders {
			if s := resp.Header(name); s != "" {
				b.WithHeader(name, s)
			}
		}

		b.WithBody(resp.Body())
		b.WithoutCompression()
		b.Write()
	})
}
