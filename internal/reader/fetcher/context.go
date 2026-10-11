package fetcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type requestContext struct {
	client    *Client
	req       *http.Request
	proxy     url.URL
	bodyClose func()
}

type ctxRequest struct{}

var requestContextKey = ctxRequest{}

func contextWithRequest(ctx context.Context, r *requestContext,
) context.Context {
	return context.WithValue(ctx, requestContextKey, r)
}

func requestFromContext(ctx context.Context) *requestContext {
	r, ok := ctx.Value(requestContextKey).(*requestContext)
	if !ok || r == nil {
		panic("fetcher: expected *requestContext")
	}

	if err := r.Validate(); err != nil {
		panic(fmt.Errorf("fetcher: invalid requestContext: %w", err))
	}
	return r
}

func (self *requestContext) Validate() error {
	if self.client == nil {
		return errors.New("nil client")
	}
	if self.req == nil {
		return errors.New("nil req")
	}
	return nil
}

func (self *requestContext) WithProxy(u *url.URL) *url.URL {
	switch u {
	case nil:
		self.proxy = url.URL{}
	default:
		self.proxy = *u
	}
	return u
}

func (self *requestContext) WithBodyClose(fn func()) *requestContext {
	self.bodyClose = fn
	return self
}

func (self *requestContext) BodyClose() {
	if self.bodyClose != nil {
		self.bodyClose()
	}
}
