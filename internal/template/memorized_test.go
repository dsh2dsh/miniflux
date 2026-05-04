package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemorized_From(t *testing.T) {
	var called int
	fn := func() bool {
		called++
		return true
	}

	var value memorized[bool]
	value.From(fn)
	assert.True(t, value.done)
	assert.True(t, value.value)
	assert.Equal(t, 1, called)

	value.From(fn)
	assert.Equal(t, 1, called)
}
