package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"

	"miniflux.app/v2/internal/config"
	"miniflux.app/v2/internal/logging"
)

var limitConnections = NewLimitPerServer()

func ExpireHostLimits(d time.Duration) { limitConnections.Expire(d) }

func newResponseSemaphore(r *RequestBuilder, rawURL string,
) (*ResponseSemaphore, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: parse %q: %w", rawURL, err)
	}
	hostname := u.Hostname()

	if err := limitConnections.Acquire(r.Context(), hostname); err != nil {
		return nil, err
	}

	//nolint:bodyclose // ResponseSemaphore.Close() it
	resp, err := r.execute(rawURL)
	return &ResponseSemaphore{
		ResponseHandler: &ResponseHandler{
			httpResponse: resp,
			clientErr:    err,
			maxBodySize:  config.Opts.HTTPClientMaxBodySize(),
		},
		release: func() { limitConnections.Release(hostname) },
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
	self := &hostLimit{connections: n, rateLimit: r}
	return self.init()
}

type hostLimit struct {
	connections int64
	rateLimit   float64

	sem  *semaphore.Weighted
	rate *rate.Limiter
	refs int

	releasedAt time.Time
}

func (self *hostLimit) init() *hostLimit {
	if self.connections > 0 {
		self.sem = semaphore.NewWeighted(self.connections)
	}
	if self.rateLimit > 0 {
		self.rate = rate.NewLimiter(rate.Limit(self.rateLimit), 1)
	}
	return self
}

func (self *hostLimit) AllowRate() bool {
	return self.rate == nil || self.rate.Allow()
}

func (self *hostLimit) WaitRate(ctx context.Context) error {
	if self.rate == nil {
		return nil
	}

	if err := self.rate.Wait(ctx); err != nil {
		return fmt.Errorf("waiting for rate limiter: %w", err)
	}
	return nil
}

func (self *hostLimit) TryAcquire() bool {
	return self.sem == nil || self.sem.TryAcquire(1)
}

func (self *hostLimit) Acquire(ctx context.Context) error {
	if self.sem == nil {
		return nil
	}

	if err := self.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("waiting for semaphore: %w", err)
	}
	return nil
}

func (self *hostLimit) Release() {
	if self.sem != nil {
		self.sem.Release(1)
	}
}

func NewLimitPerServer() *limitHosts {
	return &limitHosts{servers: map[string]*hostLimit{}}
}

type limitHosts struct {
	servers map[string]*hostLimit
	mu      sync.Mutex
}

func (self *limitHosts) Expire(d time.Duration) {
	self.mu.Lock()
	for hostname, s := range self.servers {
		if s.refs == 0 && time.Since(s.releasedAt) >= d {
			delete(self.servers, hostname)
		}
	}
	self.mu.Unlock()
}

func (self *limitHosts) Acquire(ctx context.Context, hostname string) error {
	self.mu.Lock()
	s := self.servers[hostname]
	if s == nil {
		limits := config.Opts.FindHostLimits(hostname)
		s = NewHostLimit(limits.Connections, limits.Rate)
		self.servers[hostname] = s
	}
	s.refs++
	self.mu.Unlock()

	log := logging.FromContext(ctx).With(
		slog.String("hostname", hostname),
		slog.Int64("connections", s.connections),
		slog.Float64("rate", s.rateLimit))

	rateWait := func() error {
		log.Info("max rate limit reached")
		if err := s.WaitRate(ctx); err != nil {
			return fmt.Errorf("reader/fetcher: host %q: %w", hostname, err)
		}
		log.Info("acquired rate limited connection semaphore")
		return nil
	}

	if s.TryAcquire() {
		if s.rateLimit == 0 {
			if s.connections > 0 {
				log.Debug("try acquired rate limited connection semaphore")
			}
			return nil
		}
		if s.AllowRate() {
			log.Debug("allowed rate limited connection semaphore")
			return nil
		}
		return rateWait()
	}

	log.Info("max connections limit reached")
	if err := s.Acquire(ctx); err != nil {
		return fmt.Errorf("reader/fetcher: host %q: %w", hostname, err)
	}

	if s.rateLimit == 0 {
		return nil
	}
	return rateWait()
}

func (self *limitHosts) Release(hostname string) {
	self.mu.Lock()
	s := self.servers[hostname]
	s.refs--
	if s.refs == 0 {
		s.releasedAt = time.Now()
	}
	self.mu.Unlock()
	s.Release()
}
