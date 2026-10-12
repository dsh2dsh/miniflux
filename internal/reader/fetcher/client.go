package fetcher

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"syscall"
	"time"

	"github.com/klauspost/compress/gzhttp"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
)

var ErrPrivateNetworkHost = errors.New(
	"reader/fetcher: refusing to access private network host")

var (
	defaultClient *http.Client
	clientOnce    sync.Once

	proxyFromEnvironment = http.ProxyFromEnvironment
)

type Client struct {
	rb *RequestBuilder

	httpClient *http.Client

	curl             []string
	enableKeepAlives bool
}

func (self *Client) build(rb *RequestBuilder) *Client {
	self.rb = rb
	if rb.feedProxy != nil {
		self.curl = rb.feedProxy.Curl
	}

	if self.customized() {
		self.httpClient = self.makeClient()
		return self
	}

	clientOnce.Do(func() { defaultClient = self.makeClient() })
	self.httpClient = defaultClient
	return self
}

func (self *Client) customized() bool {
	return self.rb.customized || len(self.curl) != 0
}

func (self *Client) makeClient() *http.Client {
	client := &http.Client{
		Transport:     self.transport(),
		CheckRedirect: checkRedirects,
	}
	return client
}

func (self *Client) transport() http.RoundTripper {
	if len(self.curl) != 0 {
		return &curlTransport{args: self.curl}
	}

	dialer := &net.Dialer{
		Timeout:        self.rb.clientTimeout,
		KeepAlive:      30 * time.Second,
		ControlContext: denyDialToPrivate,
	}

	transport := &http.Transport{
		Proxy:                 proxyFromClient,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       self.rb.tlsConfig(),
		TLSHandshakeTimeout:   self.rb.clientTimeout,
		DisableKeepAlives:     self.rb.customized && !self.enableKeepAlives,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: self.rb.clientTimeout,
		ExpectContinueTimeout: time.Second,

		// Setting `DialContext` disables HTTP/2, this option forces the transport
		// to try HTTP/2 regardless.
		ForceAttemptHTTP2: true,
	}

	if self.rb.disableHTTP2 {
		transport.ForceAttemptHTTP2 = false

		// https://pkg.go.dev/net/http#hdr-HTTP_2
		//
		// Programs that must disable HTTP/2 can do so by setting
		// [Transport.TLSNextProto] (for clients) or [Server.TLSNextProto] (for
		// servers) to a non-nil, empty map.
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}
	return gzhttp.Transport(transport)
}

func denyDialToPrivate(ctx context.Context, network, address string,
	_ syscall.RawConn,
) error {
	r := requestFromContext(ctx)
	if r.client.rb.allowPrivateNets || r.proxy.Host == address {
		return nil
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: split %q: %w", ErrPrivateNetworkHost, address, err)
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("%w: parse %q: %w", ErrPrivateNetworkHost, address, err)
	}

	private := addr.IsLinkLocalMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLoopback() ||
		addr.IsMulticast() ||
		addr.IsPrivate() ||
		addr.IsUnspecified() ||
		config.FetcherDeniedNetwork(addr.Unmap())

	if !private {
		return nil
	}

	reqURL := r.req.URL.String()
	ok := config.FetcherHostPermitted(address, reqURL) ||
		config.FetcherHostPermitted(host, reqURL)
	if !ok {
		return fmt.Errorf("%w: address=%q url=%q", ErrPrivateNetworkHost, address,
			reqURL)
	}
	return nil
}

func proxyFromClient(req *http.Request) (*url.URL, error) {
	r := requestFromContext(req.Context())
	if u := r.client.rb.proxy(); u != nil {
		return r.WithProxy(u), nil
	}

	u, err := proxyFromEnvironment(req)
	if err != nil {
		return nil, fmt.Errorf("fetcher: proxy from env: %w", err)
	}
	return r.WithProxy(u), nil
}

func checkRedirects(r *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}

	rc := requestFromContext(r.Context())
	if rc.client.rb.withoutRedirects {
		return http.ErrUseLastResponse
	}
	return nil
}

func (self *Client) Do(req *http.Request) (*ResponseHandler, error) {
	ctx := req.Context()
	log := logging.FromContext(ctx)
	log.Debug("Making outgoing request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Any("headers", req.Header),
		slog.Bool("without_redirects", self.rb.withoutRedirects),
		slog.Bool("use_app_client_proxy", self.rb.useClientProxy),
		slog.String("client_proxy_url", self.rb.proxyRedacted()),
		slog.Bool("ignore_tls_errors", self.rb.ignoreTLSErrors),
		slog.Bool("disable_http2", self.rb.disableHTTP2),
		slog.Bool("customized", self.rb.customized))

	hostname := req.URL.Hostname()
	if err := limits.Acquire(ctx, hostname); err != nil {
		return nil, err
	}

	rc := requestContext{client: self, req: req}
	ctx = contextWithRequest(ctx, &rc)

	ctx, cancel := context.WithTimeout(ctx, self.rb.clientTimeout)
	start := time.Now()

	//nolint:bodyclose // ResponseSemaphore.Close() it later
	resp, err := self.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		limits.Release(hostname)
		cancel()
		return nil, fmt.Errorf("fetcher: do http request: %w", err)
	}

	log.Info("Got response",
		slog.Int("status_code", resp.StatusCode),
		slog.String("status", resp.Status),
		slog.Int64("content_length", resp.ContentLength),
		slog.String("proto", resp.Proto),
		slog.String("content_type", resp.Header.Get("Content-Type")),
		slog.Duration("request_time", time.Since(start)))

	//nolint:bodyclose // ResponseSemaphore.Close() it later
	return NewResponseHandler(afterBodyClose(resp, cancel)), nil
}

func (self *Client) Request(ctx context.Context, requestURL string,
) (*ResponseHandler, error) {
	req, err := self.rb.NewRequest(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	return self.Do(req)
}

func (self *Client) Close() {
	if self.customized() {
		self.httpClient.CloseIdleConnections()
	}
}
