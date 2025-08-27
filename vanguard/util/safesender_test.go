package util

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ut "cement/unittest"
	"g53"
	"vanguard/testutil"
)

func doParallelForward(server string, sender *SafeUDPSender, name string, count int) uint32 {
	var wg sync.WaitGroup
	var errCount uint32
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			var qname *g53.Name
			if name != "" {
				qname, _ = g53.NameFromString(name)
			} else {
				qname, _ = g53.NameFromString(fmt.Sprintf("www.knet%d.cn.", index))
			}
			query := g53.MakeQuery(qname, g53.RR_A, 1024, false)
			_, _, err := sender.Query(server, query)
			if err != nil {
				atomic.AddUint32(&errCount, 1)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	return errCount
}

var defaultTimeout = 2 * time.Second

func TestSafeUDPFwderFwdLocal(t *testing.T) {
	localDNSServer := "127.0.0.1:5553"
	localServer, err := testutil.NewServer(localDNSServer)
	ut.Assert(t, err == nil, "create local echo server failed")
	go localServer.Run()
	defer localServer.Stop()

	defaultTimeout := 2 * time.Second
	sender, err := NewSafeUDPSender("", defaultTimeout)
	ut.Assert(t, err == nil, "connect to public dns server shouldn't failed")
	errCount := doParallelForward(localDNSServer, sender, "", 200)
	ut.Equal(t, errCount, uint32(0))
}

func TestSafeUDPFwderFwdPublicDNS(t *testing.T) {
	publicDNSServer := "114.114.114.114:53"
	sender, _ := NewSafeUDPSender("", defaultTimeout)
	errCount := doParallelForward(publicDNSServer, sender, "www.knet.cn.", 10)
	ut.Equal(t, errCount, uint32(0))
}

func TestSafeUDPFwderFwdNonexist(t *testing.T) {
	unreachableDNSServer := "2.2.2.2:53"
	sender, _ := NewSafeUDPSender("", defaultTimeout)
	errCount := doParallelForward(unreachableDNSServer, sender, "www.knet.cn.", 1)
	ut.Equal(t, errCount, uint32(1))
	<-time.After(time.Second * 5)
	errCount = doParallelForward(unreachableDNSServer, sender, "www.knet.cn.", 2)
	ut.Equal(t, errCount, uint32(2))
}
