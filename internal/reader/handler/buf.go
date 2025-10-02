package handler

import (
	"bytes"
	"sync"

	"miniflux.app/v2/internal/config"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

type bodyBuffer struct {
	*bytes.Buffer
}

func newBodyBuffer() bodyBuffer {
	return bodyBuffer{bufPool.Get().(*bytes.Buffer)}
}

func (self *bodyBuffer) Free() {
	if self.Buffer == nil {
		return
	}

	if self.Len() < int(config.Opts.HTTPClientMaxBodySize()) {
		self.Reset()
		bufPool.Put(self.Buffer)
	}
	self.Buffer = nil
}
