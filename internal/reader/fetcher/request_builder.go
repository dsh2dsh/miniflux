// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
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

var ProxyRotatorInstance *proxyrotator.ProxyRotator

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

func Request(ctx context.Context, requestURL string, opts ...Option,
) (*ResponseHandler, error) {
	return NewRequestBuilder(opts...).Request(ctx, requestURL)
}

type RequestBuilder struct {
	headers          http.Header
	clientProxy      *config.Proxy
	clientTimeout    time.Duration
	useClientProxy   bool
	withoutRedirects bool
	ignoreTLSErrors  bool
	disableHTTP2     bool
	proxyRotator     *proxyrotator.ProxyRotator
	feedProxyId      string
	allowPrivateNets bool

	customized bool
}

func NewRequestBuilder(opts ...Option) *RequestBuilder {
	headers := make(http.Header, 2)
	headers.Set(uaHeaderName, config.HTTPClientUserAgent())

	self := &RequestBuilder{
		headers:          headers,
		allowPrivateNets: config.FetcherAllowPrivateNetworks(),
		clientProxy:      config.ClientProxy(),
		clientTimeout:    config.HTTPClientTimeout(),
		proxyRotator:     ProxyRotatorInstance,
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
		UseCustomApplicationProxy(f.FetchViaProxy).
		WithCookie(f.Cookie).
		WithCustomFeedProxy(f.ProxyURL).
		WithUsernameAndPassword(f.Username, f.Password)
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

func (self *RequestBuilder) UseCustomApplicationProxy(value bool) *RequestBuilder {
	self.useClientProxy = value
	return self
}

func (self *RequestBuilder) WithCustomFeedProxy(id string) *RequestBuilder {
	self.feedProxyId = id
	return self
}

func (self *RequestBuilder) WithoutRedirects() *RequestBuilder {
	self.withoutRedirects = true
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
	self.allowPrivateNets = true
	return self
}

func (self *RequestBuilder) WithIntegrationDefaults() *RequestBuilder {
	if config.IntegrationAllowPrivateNetworks() {
		return self.WithPrivateNetworks()
	}
	return self
}

func (self *RequestBuilder) proxy() *config.Proxy {
	if p := config.FindProxy(self.feedProxyId); p != nil {
		return p
	}

	switch {
	case self.useClientProxy && self.clientProxy != nil:
		return self.clientProxy
	case self.proxyRotator != nil && self.proxyRotator.HasProxies():
		return config.NewProxy(self.proxyRotator.GetNextProxy())
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

func (self *RequestBuilder) Do(req *http.Request) (*ResponseHandler, error) {
	if req.Header.Get(uaHeaderName) == "" {
		req.Header.Set(uaHeaderName, config.HTTPClientUserAgent())
	}

	var client Client
	return client.build(self).Do(req)
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

func (self *RequestBuilder) Request(ctx context.Context, requestURL string,
) (*ResponseHandler, error) {
	req, err := self.NewRequest(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	return self.Do(req)
}

func (self *RequestBuilder) NewClient() (*Client, error) {
	client := (&Client{enableKeepAlives: true}).build(self)
	return client, nil
}
