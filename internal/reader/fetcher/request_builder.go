// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/proxyrotator"
)

const (
	defaultAcceptHeader = "application/xml, application/atom+xml, application/rss+xml, application/rdf+xml, application/feed+json, text/html, */*;q=0.9"
)

type RequestBuilder struct {
	ctx              context.Context
	headers          http.Header
	clientProxyURL   *url.URL
	clientTimeout    int
	useClientProxy   bool
	withoutRedirects bool
	ignoreTLSErrors  bool
	disableHTTP2     bool
	proxyRotator     *proxyrotator.ProxyRotator
	feedProxyURL     string

	customizedClient bool
}

func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		headers:        make(http.Header),
		clientProxyURL: config.Opts.HTTPClientProxyURL(),
		clientTimeout:  config.Opts.HTTPClientTimeout(),
		proxyRotator:   proxyrotator.ProxyRotatorInstance,
	}
}

func (r *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	r.ctx = ctx
	return r
}

func (r *RequestBuilder) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	r.headers.Set(key, value)
	return r
}

func (r *RequestBuilder) WithETag(etag string) *RequestBuilder {
	if etag != "" {
		r.headers.Set("If-None-Match", etag)
	}
	return r
}

func (r *RequestBuilder) WithLastModified(lastModified string) *RequestBuilder {
	if lastModified != "" {
		r.headers.Set("If-Modified-Since", lastModified)
	}
	return r
}

func (r *RequestBuilder) WithUserAgent(userAgent string, defaultUserAgent string) *RequestBuilder {
	if userAgent != "" {
		r.headers.Set("User-Agent", userAgent)
	} else {
		r.headers.Set("User-Agent", defaultUserAgent)
	}
	return r
}

func (r *RequestBuilder) WithCookie(cookie string) *RequestBuilder {
	if cookie != "" {
		r.headers.Set("Cookie", cookie)
	}
	return r
}

func (r *RequestBuilder) WithUsernameAndPassword(username, password string) *RequestBuilder {
	if username != "" && password != "" {
		r.headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	}
	return r
}

func (r *RequestBuilder) UseCustomApplicationProxyURL(value bool) *RequestBuilder {
	r.useClientProxy = value
	if value {
		r.customizedClient = true
	}
	return r
}

func (r *RequestBuilder) WithCustomFeedProxyURL(proxyURL string) *RequestBuilder {
	r.feedProxyURL = proxyURL
	if proxyURL != "" {
		r.customizedClient = true
	}
	return r
}

func (r *RequestBuilder) Timeout() time.Duration {
	return time.Duration(r.clientTimeout) * time.Second
}

func (r *RequestBuilder) WithoutRedirects() *RequestBuilder {
	r.withoutRedirects = true
	r.customizedClient = true
	return r
}

func (r *RequestBuilder) DisableHTTP2(value bool) *RequestBuilder {
	r.disableHTTP2 = value
	if value {
		r.customizedClient = true
	}
	return r
}

func (r *RequestBuilder) IgnoreTLSErrors(value bool) *RequestBuilder {
	r.ignoreTLSErrors = value
	if value {
		r.customizedClient = true
	}
	return r
}

func (r *RequestBuilder) ExecuteRequest(requestURL string) (*http.Response,
	error,
) {
	proxyURL, err := r.proxyURL()
	if err != nil {
		return nil, err
	}

	req, err := r.req(requestURL)
	if err != nil {
		return nil, err
	}

	var proxyURLRedacted string
	if proxyURL != nil {
		proxyURLRedacted = proxyURL.Redacted()
	}

	logging.FromContext(r.Context()).Info("Making outgoing request",
		slog.Bool("customized", r.customizedClient),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Any("headers", req.Header),
		slog.Bool("without_redirects", r.withoutRedirects),
		slog.Bool("use_app_client_proxy", r.useClientProxy),
		slog.String("client_proxy_url", proxyURLRedacted),
		slog.Bool("ignore_tls_errors", r.ignoreTLSErrors),
		slog.Bool("disable_http2", r.disableHTTP2))

	resp, err := r.client(proxyURL).Do(req)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: do http request: %w", err)
	}
	return resp, nil
}

func (r *RequestBuilder) proxyURL() (*url.URL, error) {
	var proxyURL *url.URL
	switch {
	case r.feedProxyURL != "":
		u, err := url.Parse(r.feedProxyURL)
		if err != nil {
			return nil, fmt.Errorf("reader/fetcher: invalid feed proxy URL %q: %w",
				r.feedProxyURL, err)
		}
		proxyURL = u
	case r.useClientProxy && r.clientProxyURL != nil:
		proxyURL = r.clientProxyURL
	case r.proxyRotator != nil && r.proxyRotator.HasProxies():
		proxyURL = r.proxyRotator.GetNextProxy()
	}
	return proxyURL, nil
}

var (
	defaultClient *http.Client
	onceClient    sync.Once
)

func (r *RequestBuilder) client(proxyURL *url.URL) *http.Client {
	if r.customizedClient {
		return r.makeClient(proxyURL)
	}
	onceClient.Do(func() { defaultClient = r.makeClient(proxyURL) })
	return defaultClient
}

func (r *RequestBuilder) makeClient(proxyURL *url.URL) *http.Client {
	client := &http.Client{
		Transport: r.transport(proxyURL),
		Timeout:   r.Timeout(),
	}

	if r.withoutRedirects {
		client.CheckRedirect = withoutRedirects
	}
	return client
}

func withoutRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (r *RequestBuilder) transport(proxyURL *url.URL) http.RoundTripper {
	dialer := &net.Dialer{Timeout: r.Timeout()}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       r.tlsConfig(),
		TLSHandshakeTimeout:   r.Timeout(),
		DisableKeepAlives:     r.customizedClient,
		IdleConnTimeout:       10 * time.Second,
		ResponseHeaderTimeout: r.Timeout(),

		// Setting `DialContext` disables HTTP/2, this option forces the transport
		// to try HTTP/2 regardless.
		ForceAttemptHTTP2: true,
	}

	if r.disableHTTP2 {
		transport.ForceAttemptHTTP2 = false

		// https://pkg.go.dev/net/http#hdr-HTTP_2
		//
		// Programs that must disable HTTP/2 can do so by setting
		// [Transport.TLSNextProto] (for clients) or [Server.TLSNextProto] (for
		// servers) to a non-nil, empty map.
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}

	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return gzhttp.Transport(transport)
}

func (r *RequestBuilder) tlsConfig() *tls.Config {
	if !r.ignoreTLSErrors {
		return nil
	}

	// We get the safe ciphers and the insecure ones if we are ignoring TLS
	// errors. This allows to connect to badly configured servers anyway.
	ciphers := slices.Concat(tls.CipherSuites(), tls.InsecureCipherSuites())
	cipherSuites := make([]uint16, len(ciphers))
	for i, cipher := range ciphers {
		cipherSuites[i] = cipher.ID
	}

	return &tls.Config{
		CipherSuites:       cipherSuites,
		InsecureSkipVerify: r.ignoreTLSErrors,
	}
}

func (r *RequestBuilder) req(requestURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: create http request: %w", err)
	}
	req.Header = r.headers.Clone()
	req.Header.Set("Accept", defaultAcceptHeader)
	return req, nil
}
