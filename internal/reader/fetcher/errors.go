package fetcher

import (
	"fmt"
	"net/http"
	"time"

	"miniflux.app/v2/internal/locale"
)

type ErrTooManyRequests struct {
	hostname   string
	retryAfter time.Time
}

var _ error = (*ErrTooManyRequests)(nil)

func NewErrTooManyRequests(hostname string, retryAfter time.Time,
) *ErrTooManyRequests {
	return &ErrTooManyRequests{
		hostname:   hostname,
		retryAfter: retryAfter,
	}
}

func (self *ErrTooManyRequests) Error() string {
	return fmt.Sprintf("host %q rate limited: 429 %s",
		self.hostname, http.StatusText(http.StatusTooManyRequests))
}

func (self *ErrTooManyRequests) RetryAfter() time.Time {
	return self.retryAfter
}

func (self *ErrTooManyRequests) Localized() *locale.LocalizedErrorWrapper {
	return locale.NewLocalizedErrorWrapper(self, "error.http_too_many_requests")
}
