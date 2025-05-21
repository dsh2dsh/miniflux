package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
)

var limitConnections = NewLimitPerServer()

func NewResponseSemaphore(ctx context.Context, r *RequestBuilder, rawURL string,
) (*ResponseSemaphore, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("reader/handler: parse %q: %w", rawURL, err)
	}
	hostname := u.Hostname()

	if err := limitConnections.Acquire(ctx, hostname); err != nil {
		return nil, err
	}

	//nolint:bodyclose // ResponseSemaphore.Close() it
	resp, err := r.WithContext(ctx).ExecuteRequest(rawURL)
	return &ResponseSemaphore{
		ResponseHandler: NewResponseHandler(resp, err),
		release:         func() { limitConnections.Release(hostname) },
	}, nil
}

type ResponseSemaphore struct {
	*ResponseHandler

	closed  bool
	release func()
}

func (self *ResponseSemaphore) Close() {
	if self.closed {
		return
	}
	self.ResponseHandler.Close()
	self.release()
	self.release = nil
	self.closed = true
}

func NewHostLimit(n int64, r float64) *hostLimit {
	return &hostLimit{
		sem:  semaphore.NewWeighted(n),
		rate: rate.NewLimiter(rate.Limit(r), 1),
	}
}

type hostLimit struct {
	sem  *semaphore.Weighted
	rate *rate.Limiter
	refs int
}

func NewLimitPerServer() *limitHosts {
	return &limitHosts{servers: map[string]*hostLimit{}}
}

type limitHosts struct {
	servers map[string]*hostLimit
	mu      sync.Mutex
}

func (self *limitHosts) Acquire(ctx context.Context, hostname string) error {
	self.mu.Lock()
	s := self.servers[hostname]
	if s == nil {
		s = NewHostLimit(
			config.Opts.ConnectionsPerServer(),
			config.Opts.RateLimitPerServer())
		self.servers[hostname] = s
	}
	s.refs++
	self.mu.Unlock()

	log := logging.FromContext(ctx).With(slog.String("hostname", hostname))
	rateWait := func() error {
		log.Info("max rate limit reached")
		if err := s.rate.Wait(ctx); err != nil {
			return fmt.Errorf("reader/fetcher: wait for rate limit host %q: %w",
				hostname, err)
		}
		log.Info("acquired rate limited connection semaphore")
		return nil
	}

	if s.sem.TryAcquire(1) {
		if s.rate.Allow() {
			log.Debug("try acquired rate limited connection semaphore")
			return nil
		}
		return rateWait()
	}

	log.Info("max connections limit reached")
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf(
			"reader/fetcher: acquire semaphore for host %q: %w", hostname, err)
	}
	return rateWait()
}

func (self *limitHosts) Release(hostname string) {
	self.mu.Lock()
	s := self.servers[hostname]
	s.refs--
	if s.refs == 0 {
		delete(self.servers, hostname)
	}
	self.mu.Unlock()
	s.sem.Release(1)
}
