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

var limits = NewLimitsPerServer()

func ExpireHostLimits(d time.Duration) { limits.Expire(d) }

type ResponseSemaphore struct {
	*ResponseHandler

	tooManyRequests bool
	retryAfter      time.Time

	closed  bool
	release func()
}

func newResponseSemaphore(r *RequestBuilder, rawURL string,
) (*ResponseSemaphore, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("reader/fetcher: parse %q: %w", rawURL, err)
	}
	hostname := u.Hostname()

	if err := limits.Acquire(r.Context(), hostname); err != nil {
		return nil, err
	}

	//nolint:bodyclose // ResponseSemaphore.Close() it
	resp, err := r.execute(rawURL)
	self := &ResponseSemaphore{
		ResponseHandler: &ResponseHandler{
			httpResponse: resp,
			clientErr:    err,
			maxBodySize:  config.Opts.HTTPClientMaxBodySize(),
		},
		release: func() { limits.Release(hostname) },
	}
	return self.init(hostname), nil
}

func (self *ResponseSemaphore) init(hostname string) *ResponseSemaphore {
	if self.Err() != nil {
		return self
	}

	if self.rateLimited() {
		retryAfter := time.Now().Add(self.parseRetryDelay())
		self.tooManyRequests = true
		self.retryAfter = limits.SetRetryAfter(hostname, retryAfter)
	}
	return self
}

func (self *ResponseSemaphore) TooManyRequests() (time.Time, bool) {
	return self.retryAfter, self.tooManyRequests
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

type limitHosts struct {
	servers map[string]*hostLimit
	mu      sync.Mutex
}

func NewLimitsPerServer() *limitHosts {
	return &limitHosts{servers: map[string]*hostLimit{}}
}

func (self *limitHosts) Acquire(ctx context.Context, hostname string) error {
	s := self.hostLimit(hostname)
	if retryAfter, ok := s.TooManyRequests(); ok {
		return NewErrTooManyRequests(hostname, retryAfter)
	}
	s.IncRefs()

	if err := self.acquire(ctx, hostname, s); err != nil {
		return err
	}

	if retryAfter, ok := s.TooManyRequests(); ok {
		s.Release()
		return NewErrTooManyRequests(hostname, retryAfter)
	}
	return nil
}

func (self *limitHosts) hostLimit(hostname string) *hostLimit {
	self.mu.Lock()
	defer self.mu.Unlock()

	s, ok := self.servers[hostname]
	if !ok {
		limits := config.Opts.FindHostLimits(hostname)
		s = NewHostLimit(limits.Connections, limits.Rate)
		self.servers[hostname] = s
	}
	return s
}

func (self *limitHosts) acquire(ctx context.Context, hostname string,
	s *hostLimit,
) error {
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
		if !s.RateLimitConfigured() {
			if s.ConnectionsLimitConfigured() {
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

	if !s.RateLimitConfigured() {
		return nil
	}
	return rateWait()
}

func (self *limitHosts) SetRetryAfter(hostname string, retryAfter time.Time,
) time.Time {
	return self.hostLimit(hostname).SetRetryAfter(retryAfter)
}

func (self *limitHosts) Release(hostname string) {
	s := self.hostLimit(hostname)
	s.Release()
}

func (self *limitHosts) Expire(d time.Duration) {
	self.mu.Lock()
	for hostname, s := range self.servers {
		if s.Expired(d) {
			delete(self.servers, hostname)
		}
	}
	self.mu.Unlock()
}

type hostLimit struct {
	connections int64
	rateLimit   float64

	sem  *semaphore.Weighted
	rate *rate.Limiter
	refs int
	mu   sync.Mutex

	tooManyRequests bool
	retryAfter      time.Time

	releasedAt time.Time
}

func NewHostLimit(n int64, r float64) *hostLimit {
	self := &hostLimit{
		connections: n,
		rateLimit:   r,
	}
	return self.init()
}

func (self *hostLimit) init() *hostLimit {
	if self.ConnectionsLimitConfigured() {
		self.sem = semaphore.NewWeighted(self.connections)
	}
	if self.RateLimitConfigured() {
		self.rate = rate.NewLimiter(rate.Limit(self.rateLimit), 1)
	}
	return self
}

func (self *hostLimit) IncRefs() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.refs++
}

func (self *hostLimit) ConnectionsLimitConfigured() bool {
	return self.connections > 0
}

func (self *hostLimit) RateLimitConfigured() bool {
	return self.rateLimit > 0
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

	self.mu.Lock()
	defer self.mu.Unlock()

	self.refs--
	if self.refs == 0 {
		self.releasedAt = time.Now()
	}
}

func (self *hostLimit) Expired(d time.Duration) bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.refs == 0 && time.Since(self.releasedAt) >= d
}

func (self *hostLimit) SetRetryAfter(retryAfter time.Time) time.Time {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.tooManyRequests || retryAfter.After(self.retryAfter) {
		self.retryAfter = retryAfter
		self.tooManyRequests = true
	}
	return self.retryAfter
}

func (self *hostLimit) TooManyRequests() (time.Time, bool) {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.retryAfter, self.tooManyRequests
}
