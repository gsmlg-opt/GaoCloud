package forwarder

import (
	"sync"
	"sync/atomic"
	"time"

	"g53"
	"g53/util"
	vutil "vanguard/util"
)

type SafeUDPFwder struct {
	fwder *vutil.SafeUDPSender

	remoteAddr   string
	lastRtt      time.Duration
	fwderTimeout time.Duration

	bearableFailInterval int64 //seconds
	lastFailTime         int64 //unix seconds format
	isDown               bool
	statusLock           sync.Mutex
}

func NewSafeUDPFwder(addr string, fwderTimeout, bearableFailInterval time.Duration) (*SafeUDPFwder, error) {
	return &SafeUDPFwder{
		remoteAddr:           addr,
		fwderTimeout:         fwderTimeout,
		bearableFailInterval: int64(bearableFailInterval.Seconds()),
	}, nil
}

func (f *SafeUDPFwder) SetQuerySource(ip string) error {
	sender, err := vutil.NewSafeUDPSender(ip, f.fwderTimeout)
	if err != nil {
		return err
	} else {
		f.fwder = sender
		return nil
	}
}

func (f *SafeUDPFwder) Forward(query *g53.Message) (*g53.Message, time.Duration, error) {
	originalQueryId := query.Header.Id
	query.Header.Id = util.GenMessageId()
	resp, rtt, err := f.fwder.Query(f.remoteAddr, query)
	atomic.StoreInt64((*int64)(&f.lastRtt), int64(rtt))
	f.checkStatus(err)
	query.Header.Id = originalQueryId
	if resp != nil {
		resp.Header.Id = originalQueryId
	}
	return resp, rtt, err
}

func (f *SafeUDPFwder) checkStatus(err error) {
	f.statusLock.Lock()
	defer f.statusLock.Unlock()
	if err == nil {
		f.lastFailTime = 0
		f.isDown = false
	} else if f.lastFailTime == 0 {
		f.lastFailTime = time.Now().Unix()
		f.isDown = false
	} else {
		f.isDown = (time.Now().Unix()-f.lastFailTime >= f.bearableFailInterval)
	}
}

func (f *SafeUDPFwder) IsDown() bool {
	f.statusLock.Lock()
	defer f.statusLock.Unlock()
	return f.isDown
}

func (f *SafeUDPFwder) GetLastRtt() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&f.lastRtt)))
}

func (f *SafeUDPFwder) RemoteAddr() string {
	return f.remoteAddr
}
