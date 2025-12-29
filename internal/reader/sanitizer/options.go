package sanitizer

import (
	"net/url"

	"github.com/dsh2dsh/bluemonday/v2"
)

type RewriteURLFunc func(t *bluemonday.Token, attr string, u *url.URL) *url.URL

type Config struct {
	RewriteURL RewriteURLFunc
}

type Option func(*Config)

func WithRewriteURL(fn RewriteURLFunc) Option {
	return func(c *Config) { c.RewriteURL = fn }
}
