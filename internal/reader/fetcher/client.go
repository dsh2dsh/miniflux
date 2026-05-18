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

var (
	defaultClient *http.Client
	onceClient    sync.Once
)

var ErrPrivateNetworkHost = errors.New(
	"reader/fetcher: refusing to access private network host")

type Client struct {
	rb *RequestBuilder

	proxy      *url.URL
	httpClient *http.Client
}

func (self *Client) Build(rb *RequestBuilder) error {
	self.rb = rb
	u, err := self.rb.proxy()
	if err != nil {
		return err
	}
	self.proxy = u

	if self.rb.customized {
		self.httpClient = self.makeClient()
	} else {
		onceClient.Do(func() { defaultClient = self.makeClient() })
		self.httpClient = defaultClient
	}
	return nil
}

func (self *Client) makeClient() *http.Client {
	client := &http.Client{
		Transport: self.transport(),
		Timeout:   self.rb.clientTimeout,
	}

	if self.rb.withoutRedirects {
		client.CheckRedirect = withoutRedirects
	}
	return client
}

func (self *Client) transport() http.RoundTripper {
	dialer := &net.Dialer{Timeout: self.rb.clientTimeout}
	if !self.rb.allowPrivateNets && self.proxy == nil {
		dialer.ControlContext = denyDialToPrivate
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       self.rb.tlsConfig(),
		TLSHandshakeTimeout:   self.rb.clientTimeout,
		DisableKeepAlives:     self.rb.customized,
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

	if self.proxy != nil {
		transport.Proxy = http.ProxyURL(self.proxy)
	}
	return gzhttp.Transport(transport)
}

func denyDialToPrivate(ctx context.Context, network, address string,
	_ syscall.RawConn,
) error {
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

func withoutRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (self *Client) Do(req *http.Request) (*ResponseHandler, error) {
	log := logging.FromContext(req.Context())
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
	if err := limits.Acquire(req.Context(), hostname); err != nil {
		return nil, err
	}

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
