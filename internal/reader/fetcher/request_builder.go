// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher // import "miniflux.app/v2/internal/reader/fetcher"

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/proxyrotator"
)

const (
	defaultAcceptHeader = "application/xml, application/atom+xml, application/rss+xml, application/rdf+xml, application/feed+json, text/html, */*;q=0.9"
	uaHeaderName        = "User-Agent"
)

func Do(req *http.Request, opts ...Option) (*ResponseHandler, error) {
	resp, err := NewRequestBuilder(opts...).Do(req)
	switch {
	case err != nil:
		return nil, err
	case resp.Err() != nil:
		resp.Close()
		return nil, resp.Err()
	}
	return resp, nil
}

func Request(requestURL string, opts ...Option) (*ResponseHandler, error) {
	return NewRequestBuilder(opts...).Request(requestURL)
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
	allowPrivateNets bool

	customized bool
}

func NewRequestBuilder(opts ...Option) *RequestBuilder {
	headers := make(http.Header, 2)
	headers.Set(uaHeaderName, config.HTTPClientUserAgent())

	self := &RequestBuilder{
		headers:          headers,
		allowPrivateNets: config.FetcherAllowPrivateNetworks(),
		clientProxyURL:   config.HTTPClientProxyURL(),
		clientTimeout:    config.HTTPClientTimeout(),
		proxyRotator:     proxyrotator.ProxyRotatorInstance,
	}

	for _, opt := range opts {
		opt(self)
	}
	return self
}

func NewRequestFeed(f *model.Feed) *RequestBuilder {
	return NewRequestBuilder().
		DisableHTTP2(f.DisableHTTP2).
		IgnoreTLSErrors(f.AllowSelfSignedCertificates).
		UseCustomApplicationProxyURL(f.FetchViaProxy).
		WithCookie(f.Cookie).
		WithCustomFeedProxyURL(f.ProxyURL).
		WithUsernameAndPassword(f.Username, f.Password)
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

func (self *RequestBuilder) WithUserAgent(userAgent string) *RequestBuilder {
	if userAgent != "" {
		self.headers.Set(uaHeaderName, userAgent)
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
		self.customized = true
	}
	return self
}

func (self *RequestBuilder) WithCustomFeedProxyURL(proxyURL string) *RequestBuilder {
	self.feedProxyURL = proxyURL
	if proxyURL != "" {
		self.customized = true
	}
	return self
}

func (self *RequestBuilder) WithoutRedirects() *RequestBuilder {
	self.withoutRedirects = true
	self.customized = true
	return self
}

func (self *RequestBuilder) DisableHTTP2(value bool) *RequestBuilder {
	self.disableHTTP2 = value
	if value {
		self.customized = true
	}
	return self
}

func (self *RequestBuilder) IgnoreTLSErrors(value bool) *RequestBuilder {
	self.ignoreTLSErrors = value
	if value {
		self.customized = true
	}
	return self
}

func (self *RequestBuilder) WithPrivateNetworks() *RequestBuilder {
	if !self.allowPrivateNets {
		self.allowPrivateNets = true
		self.customized = true
	}
	return self
}

func (self *RequestBuilder) WithIntegrationDefaults() *RequestBuilder {
	if config.IntegrationAllowPrivateNetworks() {
		return self.WithPrivateNetworks()
	}
	return self
}

func (self *RequestBuilder) proxy() (*url.URL, error) {
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

func (self *RequestBuilder) Do(req *http.Request) (*ResponseHandler, error) {
	if req.Header.Get(uaHeaderName) == "" {
		req.Header.Set(uaHeaderName, config.HTTPClientUserAgent())
	}

	var client Client
	if err := client.Build(self); err != nil {
		return nil, err
	}

	req = req.WithContext(contextWithRequest(req.Context(), req))
	return client.Do(req)
}

func (self *RequestBuilder) Request(requestURL string) (*ResponseHandler, error) {
	req, err := self.NewRequest(self.Context(), requestURL)
	if err != nil {
		return nil, err
	}
	return self.Do(req)
}

func (self *RequestBuilder) NewRequest(ctx context.Context, requestURL string,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
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

func (self *RequestBuilder) RequestWithContext(ctx context.Context,
	requestURL string,
) (*ResponseHandler, error) {
	req, err := self.WithContext(ctx).NewRequest(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	return self.Do(req)
}
