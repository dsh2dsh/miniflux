package fetcher

type Option func(rb *RequestBuilder)

func WithPrivateNetworks() Option {
	return func(rb *RequestBuilder) { rb.WithPrivateNetworks() }
}
