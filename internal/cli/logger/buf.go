package logger

import (
	"bytes"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

func NewBytesBuffer() *BytesBuffer { return new(BytesBuffer) }

type BytesBuffer struct {
	*bytes.Buffer
}

func (self *BytesBuffer) Alloc() { self.Buffer = bufPool.Get().(*bytes.Buffer) }

func (self *BytesBuffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if self.Cap() <= maxBufferSize {
		self.Reset()
		bufPool.Put(self.Buffer)
	}
	self.Buffer = nil
}
