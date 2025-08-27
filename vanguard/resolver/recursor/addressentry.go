package recursor

import (
	"math"
	"sync/atomic"
	"time"
)

type AddressEntry struct {
	addr string
	rtt  int64
}

func newAddressEntry(addr string, rtt time.Duration) *AddressEntry {
	return &AddressEntry{
		addr: addr,
		rtt:  rtt.Nanoseconds(),
	}
}

func (ae *AddressEntry) getAddr() string {
	return ae.addr
}

func (ae *AddressEntry) getRtt() time.Duration {
	return time.Duration(atomic.LoadInt64(&ae.rtt)) * time.Nanosecond
}

func (ae *AddressEntry) updateRtt(rtt time.Duration) {
	oldRtt := atomic.LoadInt64(&ae.rtt)
	newRtt := (oldRtt*7 + rtt.Nanoseconds()*3) / 10
	atomic.StoreInt64(&ae.rtt, newRtt)
}

func (ae *AddressEntry) setUnreachable() {
	ae.updateRtt(time.Nanosecond * time.Duration(math.MaxInt64))
}

func (ae *AddressEntry) String() string {
	return ae.addr
}
