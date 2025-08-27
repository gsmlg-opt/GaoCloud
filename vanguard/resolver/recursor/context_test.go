package recursor

import (
	"sync"
	"testing"

	ut "cement/unittest"
)

func TestCtxPool(t *testing.T) {
	cap := 10
	p := newRecursorCtxPool(cap)
	var getCtx []*RecursorCtx
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < cap; i++ {
		wg.Add(1)
		go func() {
			ctx := p.getCtx()
			ut.Assert(t, ctx != nil, "")
			mu.Lock()
			getCtx = append(getCtx, ctx)
			mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	for i := 0; i < cap-1; i++ {
		ut.Assert(t, getCtx[i] != getCtx[i+1], "")
	}

	for i := 0; i < 10; i++ {
		ctx := getCtx[i]
		p.putCtx(ctx)
		ut.Assert(t, p.getCtx() == ctx, "")
	}
}
