package util

import (
	ut "cement/unittest"
	"testing"
)

func TestBytePool(t *testing.T) {
	var size int = 4
	var width int = 10

	bufPool := NewBytePool(size, width)

	ut.Equal(t, bufPool.Width(), width)

	b := bufPool.Get()
	ut.Equal(t, len(b), width)
	bufPool.Put(b)

	for i := 0; i < size*2; i++ {
		bufPool.Put(make([]byte, bufPool.w))
	}
	close(bufPool.c)

	ut.Equal(t, len(bufPool.c), size)
}
