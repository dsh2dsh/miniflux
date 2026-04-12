package session

import "context"

type ctxKey struct{}

var sessionCtxKey ctxKey = struct{}{}

func With(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey, sess)
}

func FromContext(ctx context.Context) *Session {
	if sess, ok := ctx.Value(sessionCtxKey).(*Session); ok {
		return sess
	}
	return nil
}
