package filter

type Option func(c *Config)

func WithSkipAgedFilter(value bool) Option {
	return func(c *Config) { c.skipAge = value }
}
