package template

type memorized[T any] struct {
	done  bool
	value T
}

func (self *memorized[T]) From(fn func() T) T {
	if self.done {
		return self.value
	}

	self.value = fn()
	self.done = true
	return self.value
}
