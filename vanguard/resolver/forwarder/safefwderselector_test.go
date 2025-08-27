package forwarder

import (
	"testing"
	"time"

	ut "cement/unittest"
	"g53"
)

type dumpFwder struct {
	isDown     bool
	remoteAddr string
	lastRtt    time.Duration
}

func (f *dumpFwder) GetLastRtt() time.Duration {
	return f.lastRtt
}

func (f *dumpFwder) IsDown() bool {
	return f.isDown
}

func (f *dumpFwder) RemoteAddr() string {
	return f.remoteAddr
}

func (f *dumpFwder) Forward(query *g53.Message) (*g53.Message, time.Duration, error) {
	return nil, time.Second, nil
}

func (f *dumpFwder) SetQuerySource(string) error {
	return nil
}

func TestFixOrderSelector(t *testing.T) {
	var f1 dumpFwder
	var f2 dumpFwder
	var f3 dumpFwder

	s := newFixOrderSelector([]SafeFwder{&f1, &f2, &f3})
	ut.Assert(t, s.SelectFwder() == &f1, "first fwder should be selected")
	f1.isDown = true
	ut.Assert(t, s.SelectFwder() == &f2, "second fwder should be selected")
	f2.isDown = true
	ut.Assert(t, s.SelectFwder() == &f3, "last fwder should be selected")
	f2.isDown = false
	ut.Assert(t, s.SelectFwder() == &f2, "second fwder should be selected")
	f1.isDown = false
	ut.Assert(t, s.SelectFwder() == &f1, "second fwder should be selected")
	f1.isDown = true
	f2.isDown = true
	ut.Assert(t, s.SelectFwder() == &f3, "second fwder should be selected")
	f3.isDown = true
	ut.Assert(t, s.SelectFwder() == nil, "second fwder should be selected")
}

func TestRoundRobinSelector(t *testing.T) {
	var f1 dumpFwder
	var f2 dumpFwder
	var f3 dumpFwder

	s := newRoundRobinSelector([]SafeFwder{&f1, &f2, &f3})
	ut.Assert(t, s.SelectFwder() == &f1, "first fwder should be selected")
	ut.Assert(t, s.SelectFwder() == &f2, "second fwder should be selected")
	ut.Assert(t, s.SelectFwder() == &f3, "last fwder should be selected")
	ut.Assert(t, s.SelectFwder() == &f1, "last fwder should be selected")

	f2.isDown = true
	ut.Assert(t, s.SelectFwder() == &f3, "last fwder should be selected")
	f1.isDown = true
	f2.isDown = false
	ut.Assert(t, s.SelectFwder() == &f2, "second fwder should be selected")
	f3.isDown = true
	f1.isDown = false
	ut.Assert(t, s.SelectFwder() == &f1, "first fwder should be selected")
	f3.isDown = false
	ut.Assert(t, s.SelectFwder() == &f2, "second fwder should be selected")
	ut.Assert(t, s.SelectFwder() == &f3, "last fwder should be selected")
	f1.isDown = true
	f2.isDown = true
	f3.isDown = true
	ut.Assert(t, s.SelectFwder() == nil, "no fwder is available")
}

func TestRttSelector(t *testing.T) {
	var f1 dumpFwder
	var f2 dumpFwder
	var f3 dumpFwder

	s := newRttBasedSelector([]SafeFwder{&f1, &f2, &f3})
	f1.lastRtt = time.Second
	f2.lastRtt = 2 * time.Second
	f3.lastRtt = 3 * time.Second
	ut.Assert(t, s.SelectFwder() == &f1, "first fwder should be selected")
	ut.Assert(t, s.SelectFwder() == &f1, "last fwder should be selected")
	f1.lastRtt = 4 * time.Second
	ut.Assert(t, s.SelectFwder() == &f2, "last fwder should be selected")
	f1.lastRtt = time.Second
	ut.Assert(t, s.SelectFwder() == &f1, "last fwder should be selected")

	f1.isDown = true
	f2.isDown = true
	f3.isDown = true
	ut.Assert(t, s.SelectFwder() == nil, "no fwder is available")
}
