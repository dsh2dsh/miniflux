package fetcher

type Option func(rb *RequestBuilder)

func WithPrivateNetworks() Option {
	return func(rb *RequestBuilder) { rb.WithPrivateNetworks() }
}

func WithIntegrationDefaults() Option {
	return func(rb *RequestBuilder) { rb.WithIntegrationDefaults() }
}
