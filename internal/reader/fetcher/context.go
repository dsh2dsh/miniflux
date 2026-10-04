package fetcher

import (
	"context"
	"net/http"
	"net/url"
)

type (
	ctxClient  struct{}
	ctxProxy   struct{}
	ctxRequest struct{}
)

var (
	clientContextKey  = ctxClient{}
	proxyContextKey   = ctxProxy{}
	requestContextKey = ctxRequest{}
)

func contextWithClient(ctx context.Context, c *Client) context.Context {
	return context.WithValue(ctx, clientContextKey, c)
}

func clientFromContext(ctx context.Context) *Client {
	if b, ok := ctx.Value(clientContextKey).(*Client); ok {
		return b
	}
	return nil
}

func contextWithProxy(ctx context.Context, u *url.URL) context.Context {
	return context.WithValue(ctx, proxyContextKey, u)
}

func proxyFromContext(ctx context.Context) *url.URL {
	if u, ok := ctx.Value(proxyContextKey).(*url.URL); ok {
		return u
	}
	return nil
}

func updateContextProxy(ctx context.Context, u *url.URL) *url.URL {
	proxy := proxyFromContext(ctx)
	if proxy == nil {
		return u
	}

	switch u {
	case nil:
		*proxy = url.URL{}
	default:
		*proxy = *u
	}
	return u
}

func contextWithRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, req)
}

func requestFromContext(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok {
		return req
	}
	return nil
}
