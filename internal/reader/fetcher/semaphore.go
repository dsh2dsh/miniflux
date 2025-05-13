package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"golang.org/x/sync/semaphore"

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

func NewWeightedRefs(n int64) *weightedRefs {
	return &weightedRefs{Weighted: semaphore.NewWeighted(n)}
}

type weightedRefs struct {
	*semaphore.Weighted
	refs int
}

func NewLimitPerServer() *limitHosts {
	return &limitHosts{servers: map[string]*weightedRefs{}}
}

type limitHosts struct {
	servers map[string]*weightedRefs
	mu      sync.Mutex
}

func (self *limitHosts) Acquire(ctx context.Context, hostname string) error {
	self.mu.Lock()
	s := self.servers[hostname]
	if s == nil {
		s = NewWeightedRefs(int64(config.Opts.ConnectionsPerServer()))
		self.servers[hostname] = s
	}
	s.refs++
	self.mu.Unlock()

	log := logging.FromContext(ctx).With(slog.String("hostname", hostname))
	if s.TryAcquire(1) {
		log.Debug("try acquired connection semaphore")
		return nil
	}

	log.Info("max connections limit reached")
	if err := s.Acquire(ctx, 1); err != nil {
		return fmt.Errorf(
			"reader/handler: acquire semaphore for host %q: %w", hostname, err)
	}

	log.Info("acquired connection semaphore")
	return nil
}

func (self *limitHosts) Release(hostname string) {
	self.mu.Lock()
	s := self.servers[hostname]
	s.refs--
	if s.refs == 0 {
		delete(self.servers, hostname)
	}
	self.mu.Unlock()
	s.Release(1)
}
