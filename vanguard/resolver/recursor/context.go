package recursor

import (
	"sync"
	"time"

	"g53"
	"vanguard/util"
)

type RecursorCtx struct {
	sender        *util.SafeUDPSender
	question      *g53.Question
	clientAddress string
	depth         uint32
	startTime     time.Time
	nameServers   []*NameServer
}

func (ctx *RecursorCtx) init(queryTimeout time.Duration, querySource string, clientAddress string, question *g53.Question, nameServers []*NameServer) {
	if ctx.sender == nil || ctx.sender.GetQuerySource() != querySource {
		sender, _ := util.NewSafeUDPSender(querySource, queryTimeout)
		ctx.sender = sender
	}
	ctx.question = question
	ctx.clientAddress = clientAddress
	ctx.depth = 0
	ctx.startTime = time.Now()
	ctx.nameServers = nameServers
}

type RecursorCtxPool struct {
	ctxes []*RecursorCtx
	mu    sync.Mutex
}

func newRecursorCtxPool(max int) *RecursorCtxPool {
	ctxes := make([]*RecursorCtx, max)
	for i := 0; i < max; i++ {
		ctxes[i] = &RecursorCtx{}
	}

	return &RecursorCtxPool{
		ctxes: ctxes,
	}
}

func (p *RecursorCtxPool) getCtx() *RecursorCtx {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, ctx := range p.ctxes {
		if ctx != nil {
			p.ctxes[i] = nil
			return ctx
		}
	}
	return nil
}

func (p *RecursorCtxPool) putCtx(ctx *RecursorCtx) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, ctx_ := range p.ctxes {
		if ctx_ == nil {
			p.ctxes[i] = ctx
			break
		}
	}
}
