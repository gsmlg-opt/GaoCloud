package forwarder

import (
	"errors"
	"sync"
	"time"

	"g53"
)

var (
	ErrAllFwderIsDown = errors.New("no available forwarder left")
)

const maxRetryCount = 2

type FwderGroup struct {
	selector      FwderSelector
	lastFwderLock sync.RWMutex
	lastFwder     SafeFwder
}

func NewFwderGroup(selector FwderSelector) *FwderGroup {
	return &FwderGroup{
		selector: selector,
	}
}

func (g *FwderGroup) Forward(query *g53.Message) (resp *g53.Message, rtt time.Duration, err error) {
	for i := 0; i < maxRetryCount; i++ {
		resp, rtt, err = g.forwardOnce(query)
		if err == nil {
			return
		}
	}
	return
}

func (g *FwderGroup) forwardOnce(query *g53.Message) (*g53.Message, time.Duration, error) {
	fwder := g.selector.SelectFwder()

	g.lastFwderLock.Lock()
	g.lastFwder = fwder
	g.lastFwderLock.Unlock()

	if fwder == nil {
		return nil, 0, ErrAllFwderIsDown
	} else {
		return fwder.Forward(query)
	}
}

func (g *FwderGroup) SetQuerySource(ip string) error {
	return g.selector.SetQuerySource(ip)
}

func (g *FwderGroup) GetLastRtt() time.Duration {
	g.lastFwderLock.RLock()
	defer g.lastFwderLock.RUnlock()
	if g.lastFwder != nil {
		return g.lastFwder.GetLastRtt()
	} else {
		return 0
	}
}

func (g *FwderGroup) IsDown() bool {
	return g.selector.HasUpFwder() == false
}

func (g *FwderGroup) RemoteAddr() string {
	g.lastFwderLock.RLock()
	defer g.lastFwderLock.RUnlock()
	if g.lastFwder != nil {
		return g.lastFwder.RemoteAddr()
	} else {
		return g.selector.GetFwders()[0].RemoteAddr()
	}
}
