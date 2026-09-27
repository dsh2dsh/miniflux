package fetcher

import (
	"context"
	"net/http"
)

type (
	ctxClient  struct{}
	ctxRequest struct{}
)

var (
	clientContextKey  = ctxClient{}
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

func contextWithRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, req)
}

func requestFromContext(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok {
		return req
	}
	return nil
}
