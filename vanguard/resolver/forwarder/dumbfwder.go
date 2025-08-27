package forwarder

import (
	"errors"
	"time"

	"g53"
)

type DumbFwder struct {
	LastRtt    time.Duration
	remoteAddr string
	Down       bool
	GetError   bool
	Response   *g53.Message
}

func NewDumbFwder(addr string) *DumbFwder {
	return &DumbFwder{
		remoteAddr: addr,
	}
}

func (f *DumbFwder) RemoteAddr() string {
	return f.remoteAddr
}

func (f *DumbFwder) GetLastRtt() time.Duration {
	return f.LastRtt
}

func (f *DumbFwder) IsDown() bool {
	return f.Down
}

func (f *DumbFwder) SetQuerySource(ip string) error {
	return nil
}

func (f *DumbFwder) Forward(*g53.Message) (*g53.Message, time.Duration, error) {
	if f.GetError {
		return nil, f.LastRtt, errors.New("timeout")
	} else {
		return f.Response, f.LastRtt, nil
	}
}

func BuildDumbViewFwder(view string, zoneAndFwders map[string]*DumbFwder) *ViewFwderMgr {
	viewFwder := newViewFwder()
	for zone, fwder := range zoneAndFwders {
		viewFwder.addZoneFwder(zone, newZoneFwder(matchSubdomain, fwder))
	}

	viewFwderMgr := &ViewFwderMgr{
		fwders: make(map[string]*ViewFwder),
	}
	viewFwderMgr.fwders[view] = viewFwder
	return viewFwderMgr
}
