package fetcher

import (
	"context"
	"net/http"
)

type ctxRequest struct{}

var requestContextKey = ctxRequest{}

func contextWithRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, req)
}

func requestFromContext(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok {
		return req
	}
	return nil
}
