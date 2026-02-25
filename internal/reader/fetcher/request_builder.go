// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
)

const (
	defaultAcceptHeader = "application/xml, application/atom+xml, application/rss+xml, application/rdf+xml, application/feed+json, text/html, */*;q=0.9"
)

var errDialToPrivate = errors.New("reader/fetcher: protect private network")

func NewRequestDiscovery(d *model.SubscriptionDiscoveryRequest,
) *RequestBuilder {
	return NewRequestBuilder().
		DisableHTTP2(d.DisableHTTP2).
		IgnoreTLSErrors(d.AllowSelfSignedCertificates).
		UseCustomApplicationProxyURL(d.FetchViaProxy).
		WithCookie(d.Cookie).
		WithCustomFeedProxyURL(d.ProxyURL).
		WithUserAgent(d.UserAgent, config.HTTPClientUserAgent()).
		WithUsernameAndPassword(d.Username, d.Password)
}

func NewRequestFeed(f *model.Feed) *RequestBuilder {
	return NewRequestBuilder().
		DisableHTTP2(f.DisableHTTP2).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		WithCookie(f.Cookie).
		WithCustomFeedProxyURL(f.ProxyURL).
		WithUserAgent(f.UserAgent, config.HTTPClientUserAgent()).
		WithUsernameAndPassword(f.Username, f.Password)
}

func Request(requestURL string) (*ResponseSemaphore, error) {
	return NewRequestBuilder().Request(requestURL)
}

func RequestFeedCreation(r *model.FeedCreationRequest) (*ResponseSemaphore,
	error,
) {
	return NewRequestBuilder().
		DisableHTTP2(r.DisableHTTP2).
		IgnoreTLSErrors(r.AllowSelfSignedCertificates).
		UseCustomApplicationProxyURL(r.FetchViaProxy).
		WithCookie(r.Cookie).
		WithCustomFeedProxyURL(r.ProxyURL).
		WithUserAgent(r.UserAgent, config.HTTPClientUserAgent()).
		WithUsernameAndPassword(r.Username, r.Password).
		Request(r.FeedURL)
}

type RequestBuilder struct {
	ctx              context.Context
	headers          http.Header
	clientProxyURL   *url.URL
	clientTimeout    time.Duration
	useClientProxy   bool
	withoutRedirects bool
	ignoreTLSErrors  bool
	disableHTTP2     bool
	proxyRotator     *proxyrotator.ProxyRotator
	feedProxyURL     string
	denyPrivateNets  bool

	customizedClient bool
}

func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		headers:        make(http.Header),
		clientProxyURL: config.HTTPClientProxyURL(),
		clientTimeout:  config.HTTPClientTimeout(),
		proxyRotator:   proxyrotator.ProxyRotatorInstance,
	}
}

func (self *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	self.ctx = ctx
	return self
}

func (self *RequestBuilder) Context() context.Context {
	if self.ctx != nil {
		return self.ctx
	}
	return context.Background()
}

func (self *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	self.headers.Set(key, value)
	return self
}

func (self *RequestBuilder) WithETag(etag string) *RequestBuilder {
	if etag != "" {
		self.headers.Set("If-None-Match", etag)
	}
	return self
}

func (self *RequestBuilder) WithLastModified(lastModified string) *RequestBuilder {
	if lastModified != "" {
		self.headers.Set("If-Modified-Since", lastModified)
	}
	return self
}

func (self *RequestBuilder) WithUserAgent(userAgent, defaultUserAgent string) *RequestBuilder {
	if userAgent != "" {
		self.headers.Set("User-Agent", userAgent)
	} else {
		self.headers.Set("User-Agent", defaultUserAgent)
	}
	return self
}

func (self *RequestBuilder) WithCookie(cookie string) *RequestBuilder {
	if cookie != "" {
		self.headers.Set("Cookie", cookie)
	}
	return self
}

func (self *RequestBuilder) WithUsernameAndPassword(username, password string) *RequestBuilder {
	if username != "" && password != "" {
		self.headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	}
	return self
}

func (self *RequestBuilder) UseCustomApplicationProxyURL(value bool) *RequestBuilder {
	self.useClientProxy = value
	if value {
		self.customizedClient = true
	}
	return self
}

func (self *RequestBuilder) WithCustomFeedProxyURL(proxyURL string) *RequestBuilder {
	self.feedProxyURL = proxyURL
	if proxyURL != "" {
		self.customizedClient = true
	}
	return self
}

func (self *RequestBuilder) Timeout() time.Duration { return self.clientTimeout }

func (self *RequestBuilder) WithoutRedirects() *RequestBuilder {
	self.withoutRedirects = true
	self.customizedClient = true
	return self
}

func (self *RequestBuilder) DisableHTTP2(value bool) *RequestBuilder {
	self.disableHTTP2 = value
	if value {
		self.customizedClient = true
	}
	return self
}

func (self *RequestBuilder) IgnoreTLSErrors(value bool) *RequestBuilder {
	self.ignoreTLSErrors = value
	if value {
		self.customizedClient = true
	}
	return self
}

func (self *RequestBuilder) WithDenyPrivateNets(value bool) *RequestBuilder {
	self.denyPrivateNets = value
	self.customizedClient = value
	return self
}

func (self *RequestBuilder) execute(requestURL string) (*http.Response,
	error,
) {
	proxyURL, err := self.proxyURL()
	if err != nil {
		return nil, err
	}

	req, err := self.req(requestURL)
	if err != nil {
		return nil, err
	}

	var proxyURLRedacted string
	if proxyURL != nil {
		proxyURLRedacted = proxyURL.Redacted()
	}

	log := logging.FromContext(self.Context())
	log.Debug("Making outgoing request",
		slog.Bool("customized", self.customizedClient),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Any("headers", req.Header),
		slog.Bool("without_redirects", self.withoutRedirects),
		slog.Bool("use_app_client_proxy", self.useClientProxy),
		slog.String("client_proxy_url", proxyURLRedacted),
		slog.Bool("ignore_tls_errors", self.ignoreTLSErrors),
		slog.Bool("disable_http2", self.disableHTTP2))

	start := time.Now()
	resp, err := self.client(proxyURL).Do(req)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: do http request: %w", err)
	}

	log.Info("Got response",
		slog.Int("status_code", resp.StatusCode),
		slog.String("status", resp.Status),
		slog.Int64("content_length", resp.ContentLength),
		slog.String("proto", resp.Proto),
		slog.String("content_type", resp.Header.Get("Content-Type")),
		slog.Duration("request_time", time.Since(start)))
	return resp, nil
}

func (self *RequestBuilder) proxyURL() (*url.URL, error) {
	var proxyURL *url.URL
	switch {
	case self.feedProxyURL != "":
		u, err := url.Parse(self.feedProxyURL)
		if err != nil {
			return nil, fmt.Errorf("reader/fetcher: invalid feed proxy URL %q: %w",
				self.feedProxyURL, err)
		}
		proxyURL = u
	case self.useClientProxy && self.clientProxyURL != nil:
		proxyURL = self.clientProxyURL
	case self.proxyRotator != nil && self.proxyRotator.HasProxies():
		proxyURL = self.proxyRotator.GetNextProxy()
	}
	return proxyURL, nil
}

var (
	defaultClient *http.Client
	onceClient    sync.Once
)

func (self *RequestBuilder) client(proxyURL *url.URL) *http.Client {
	if self.customizedClient {
		return self.makeClient(proxyURL)
	}
	onceClient.Do(func() { defaultClient = self.makeClient(proxyURL) })
	return defaultClient
}

func (self *RequestBuilder) makeClient(proxyURL *url.URL) *http.Client {
	client := &http.Client{
		Transport: self.transport(proxyURL),
		Timeout:   self.Timeout(),
	}

	if self.withoutRedirects {
		client.CheckRedirect = withoutRedirects
	}
	return client
}

func withoutRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (self *RequestBuilder) transport(proxyURL *url.URL) http.RoundTripper {
	dialer := &net.Dialer{Timeout: self.Timeout()}
	if self.denyPrivateNets {
		dialer.Control = denyDialToPrivate
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       self.tlsConfig(),
		TLSHandshakeTimeout:   self.Timeout(),
		DisableKeepAlives:     self.customizedClient,
		IdleConnTimeout:       10 * time.Second,
		ResponseHeaderTimeout: self.Timeout(),

		// Setting `DialContext` disables HTTP/2, this option forces the transport
		// to try HTTP/2 regardless.
		ForceAttemptHTTP2: true,
	}

	if self.disableHTTP2 {
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

func denyDialToPrivate(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: split %q: %w", errDialToPrivate, address, err)
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: parse %q: %w", errDialToPrivate, address, err)
	}

	private := addr.IsLinkLocalMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLoopback() ||
		addr.IsMulticast() ||
		addr.IsPrivate() ||
		addr.IsUnspecified()

	if private {
		return fmt.Errorf("%w: access denied: %s", errDialToPrivate, address)
	}
	return nil
}

func (self *RequestBuilder) tlsConfig() *tls.Config {
	if !self.ignoreTLSErrors {
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
		InsecureSkipVerify: self.ignoreTLSErrors,
	}
}

func (self *RequestBuilder) req(requestURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(self.Context(), http.MethodGet,
		requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: create http request: %w", err)
	}
	req.Header = self.headers.Clone()
	// Set default Accept header if not already set. Note that for the media proxy
	// requests, we need to forward the browser Accept header.
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", defaultAcceptHeader)
	}
	return req, nil
}

func (self *RequestBuilder) Request(requestURL string) (*ResponseSemaphore,
	error,
) {
	return newResponseSemaphore(self, requestURL)
}

func (self *RequestBuilder) RequestWithContext(ctx context.Context,
	requestURL string,
) (*ResponseSemaphore, error) {
	return newResponseSemaphore(self.WithContext(ctx), requestURL)
}
