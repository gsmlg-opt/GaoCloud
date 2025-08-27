package bitmap

import (
	ut "cement/unittest"
	"testing"
)

func TestBitmap(t *testing.T) {
	bm := New(50)
	ut.Equal(t, bm.Len(), 56)

	bm.Set(30, true)
	ut.Assert(t, bm.Get(30), "")
	ut.Assert(t, !bm.Get(20), "")

	bm.Set(30, false)
	ut.Assert(t, !bm.Get(30), "")
}
