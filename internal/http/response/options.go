package response

import "net/http"

type Option func(b *Builder)

func WithHeader(key, value string) Option {
	return func(b *Builder) { b.WithHeader(key, value) }
}

func WithStatus(statusCode int) Option {
	return func(b *Builder) { b.WithStatus(statusCode) }
}

func WithStatusAccepted() Option {
	return func(b *Builder) { b.WithStatus(http.StatusAccepted) }
}

func WithStatusCreated() Option {
	return func(b *Builder) { b.WithStatus(http.StatusCreated) }
}

func WithStatusNoContent() Option {
	return func(b *Builder) { b.WithStatus(http.StatusNoContent) }
}
