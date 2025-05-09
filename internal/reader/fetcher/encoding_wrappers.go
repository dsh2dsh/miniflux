package fetcher

import (
	"compress/gzip"
	"io"

	"github.com/andybalholm/brotli"
)

type brotliReadCloser struct {
	body         io.ReadCloser
	brotliReader io.Reader
}

func NewBrotliReadCloser(body io.ReadCloser) *brotliReadCloser {
	return &brotliReadCloser{
		body:         body,
		brotliReader: brotli.NewReader(body),
	}
}

//nolint:wrapcheck // no reason wrap io.Reader
func (b *brotliReadCloser) Read(p []byte) (n int, err error) {
	return b.brotliReader.Read(p)
}

//nolint:wrapcheck // no reason wrap io.Closer
func (b *brotliReadCloser) Close() error {
	return b.body.Close()
}

type gzipReadCloser struct {
	body       io.ReadCloser
	gzipReader io.Reader
	gzipErr    error
}

func NewGzipReadCloser(body io.ReadCloser) *gzipReadCloser {
	return &gzipReadCloser{body: body}
}

//nolint:wrapcheck // no reason wrap io.Reader
func (gz *gzipReadCloser) Read(p []byte) (n int, err error) {
	if gz.gzipReader == nil {
		if gz.gzipErr == nil {
			gz.gzipReader, gz.gzipErr = gzip.NewReader(gz.body)
		}
		if gz.gzipErr != nil {
			return 0, gz.gzipErr
		}
	}
	return gz.gzipReader.Read(p)
}

//nolint:wrapcheck // no reason wrap io.Closer
func (gz *gzipReadCloser) Close() error {
	return gz.body.Close()
}
