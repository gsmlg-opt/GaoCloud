package util

import (
	"testing"
	"time"

	ut "cement/unittest"
	"g53"
)

func TestUDPFwder(t *testing.T) {
	publicDNSServer := "114.114.114.114:53"
	timeout := 2 * time.Second
	sender, err := NewUDPSender("", timeout)
	ut.Assert(t, err == nil, "connect to public dns server shouldn't failed")

	render := g53.NewMsgRender()
	qname, _ := g53.NameFromString("www.knet.cn.")
	query := g53.MakeQuery(qname, g53.RR_A, 1024, false)
	expectMaxRTT := 5 * time.Second
	for i := 0; i < 10; i++ {
		resp, rtt, err := sender.Query(publicDNSServer, render, query)
		ut.Assert(t, err == nil, "query knet cn shouldn't failed")
		ut.Assert(t, resp != nil, "query knet cn should get response")
		ut.Assert(t, rtt < expectMaxRTT, "rtt should n't bigger than 10 second")
	}

	unreachableServer := "114.114.114.114:5333"
	resp, rtt, err := sender.Query(unreachableServer, render, query)
	ut.Assert(t, err != nil, "server isn't reachable")
	ut.Assert(t, resp == nil, "no response should returned")
	ut.Equal(t, rtt, timeout)

	timeout = time.Nanosecond
	sender, _ = NewUDPSender("", timeout)
	resp, rtt, err = sender.Query(publicDNSServer, render, query)
	ut.Assert(t, err != nil, "timeout is too short")
	ut.Assert(t, resp == nil, "no response should returned")
	ut.Equal(t, rtt, timeout)
}
