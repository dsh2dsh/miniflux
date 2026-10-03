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
	onceClient    sync.Once

	proxyFromEnvironment = http.ProxyFromEnvironment
)

type Client struct {
	rb *RequestBuilder

	proxy      *config.Proxy
	httpClient *http.Client
	customized bool

	allowPrivateNets bool
	enableKeepAlives bool
	withoutRedirects bool
}

func (self *Client) build(rb *RequestBuilder) *Client {
	self.rb = rb
	self.allowPrivateNets = rb.allowPrivateNets
	self.withoutRedirects = rb.withoutRedirects

	self.proxy = rb.proxy()

	self.customized = rb.customized
	if self.customized {
		self.httpClient = self.makeClient()
		return self
	}

	onceClient.Do(func() { defaultClient = self.makeClient() })
	self.httpClient = defaultClient
	return self
}

func (self *Client) makeClient() *http.Client {
	client := &http.Client{
		Transport:     self.transport(),
		CheckRedirect: checkRedirects,
		Timeout:       self.rb.clientTimeout,
	}
	return client
}

func (self *Client) transport() http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:        self.rb.clientTimeout,
		ControlContext: denyDialToPrivate,
	}

	transport := &http.Transport{
		Proxy:                 proxyFromClient,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       self.rb.tlsConfig(),
		TLSHandshakeTimeout:   self.rb.clientTimeout,
		DisableKeepAlives:     self.rb.customized && !self.enableKeepAlives,
		MaxIdleConns:          100,
		IdleConnTimeout:       10 * time.Second,
		ResponseHeaderTimeout: self.rb.clientTimeout,

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
	c := clientFromContext(ctx)
	if c != nil && (c.allowPrivateNets || c.proxy != nil) {
		return nil
	}

	if u := proxyFromContext(ctx); u != nil && u.Host == address {
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

	var reqURL string
	if req := requestFromContext(ctx); req != nil {
		reqURL = req.URL.String()
	}

	ok := config.FetcherHostPermitted(address, reqURL) ||
		config.FetcherHostPermitted(host, reqURL)
	if !ok {
		return fmt.Errorf("%w: address=%q url=%q", ErrPrivateNetworkHost, address,
			reqURL)
	}
	return nil
}

func proxyFromClient(req *http.Request) (*url.URL, error) {
	c := clientFromContext(req.Context())
	if c != nil && c.proxy != nil {
		return c.proxy.URL(), nil
	}

	u, err := proxyFromEnvironment(req)
	if err != nil {
		return nil, fmt.Errorf("fetcher: proxy from env: %w", err)
	} else if u == nil {
		return nil, nil
	}

	if p := proxyFromContext(req.Context()); p != nil {
		*p = *u
	}
	return u, nil
}

func checkRedirects(r *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}

	c := clientFromContext(r.Context())
	if c != nil && c.withoutRedirects {
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
		slog.String("client_proxy_url", self.proxyRedacted()),
		slog.Bool("ignore_tls_errors", self.rb.ignoreTLSErrors),
		slog.Bool("disable_http2", self.rb.disableHTTP2),
		slog.Bool("customized", self.rb.customized))

	hostname := req.URL.Hostname()
	if err := limits.Acquire(ctx, hostname); err != nil {
		return nil, err
	}

	var proxy url.URL
	req = req.WithContext(self.context(ctx, req, &proxy))
	start := time.Now()

	//nolint:bodyclose // ResponseSemaphore.Close() it later
	resp, err := self.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("reader/fetcher: do http request: %w", err)
	} else {
		log.Info("Got response",
			slog.Int("status_code", resp.StatusCode),
			slog.String("status", resp.Status),
			slog.Int64("content_length", resp.ContentLength),
			slog.String("proto", resp.Proto),
			slog.String("content_type", resp.Header.Get("Content-Type")),
			slog.Duration("request_time", time.Since(start)))
	}
	return NewResponseHandler(hostname, resp, err), nil
}

func (self *Client) proxyRedacted() string {
	if self.proxy != nil {
		return self.proxy.Redacted()
	}
	return ""
}

func (self *Client) context(ctx context.Context, req *http.Request,
	proxy *url.URL,
) context.Context {
	ctx = contextWithClient(ctx, self)
	ctx = contextWithRequest(ctx, req)

	if self.proxy == nil {
		ctx = contextWithProxy(ctx, proxy)
	}
	return ctx
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
	if self.customized {
		self.httpClient.CloseIdleConnections()
	}
}
