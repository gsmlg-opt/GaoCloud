package recursor

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"g53"
)

var errNameServerNoAddr = errors.New("name server doesn't have any addr")
var errUnknownNameServer = errors.New("name server is unknown")
var errAddrIsUnknown = errors.New("addr is unknown")

type NameServer struct {
	zone *g53.Name
	name *g53.Name
	addr string
	rtt  time.Duration
}

func (ns *NameServer) String() string {
	return fmt.Sprintf("[%s ns %s A %s]", ns.zone.String(true), ns.name.String(true), ns.addr)
}

type NameServerEntry struct {
	name       *g53.Name
	addrEntrys []*AddressEntry
	expireTime time.Time
	trustLevel TrustLevel
}

func newNameServerEntry(name *g53.Name, ttl time.Duration, addrs []string, trustLevel TrustLevel) *NameServerEntry {
	if len(addrs) == 0 {
		panic(errNameServerNoAddr.Error())
	}

	ns := &NameServerEntry{
		name:       name,
		expireTime: time.Now().Add(ttl),
		trustLevel: trustLevel,
	}
	for _, addr := range addrs {
		ns.addServer(addr, time.Duration(rand.Intn(10))*time.Nanosecond)
	}

	return ns
}

func (ns *NameServerEntry) addServer(addr string, rtt time.Duration) {
	ns.addrEntrys = append(ns.addrEntrys, newAddressEntry(addr, rtt))
}

func (ns *NameServerEntry) getNameServers(zone *g53.Name) []*NameServer {
	nameServers := make([]*NameServer, len(ns.addrEntrys))
	for i := 0; i < len(ns.addrEntrys); i++ {
		nameServers[i] = &NameServer{
			zone: zone,
			name: ns.name,
			addr: ns.addrEntrys[i].addr,
			rtt:  ns.addrEntrys[i].getRtt(),
		}
	}
	return nameServers
}

func (ns *NameServerEntry) selectNameServer() *NameServer {
	minRtt := ns.addrEntrys[0].getRtt()
	selectEntry := ns.addrEntrys[0]
	for i := 1; i < len(ns.addrEntrys); i++ {
		rtt := ns.addrEntrys[i].getRtt()
		if rtt < minRtt {
			minRtt = rtt
			selectEntry = ns.addrEntrys[i]
		}
	}

	return &NameServer{
		name: ns.name,
		addr: selectEntry.addr,
		rtt:  minRtt,
	}
}

func (ns *NameServerEntry) updateRtt(nameServer *NameServer, rtt time.Duration) error {
	for _, server := range ns.addrEntrys {
		if server.addr == nameServer.addr {
			server.updateRtt(rtt)
			return nil
		}
	}
	return errAddrIsUnknown
}

func (nse *NameServerEntry) isExpired() bool {
	return nse.expireTime.Before(time.Now())
}

func (ns *NameServerEntry) String() string {
	return fmt.Sprintf("zone %s, addresses %v", ns.name.String(false), ns.addrEntrys)
}

type NameServerManager struct {
	nsEntrys map[uint32]*NameServerEntry
	lock     sync.RWMutex
}

func newNameServerManager() *NameServerManager {
	return &NameServerManager{
		nsEntrys: make(map[uint32]*NameServerEntry),
	}
}

func (ns *NameServerManager) addNameServer(name *g53.Name, ttl time.Duration, addrs []string, trustLevel TrustLevel) {
	e := newNameServerEntry(name, ttl, addrs, trustLevel)
	key := name.Hash(false)
	ns.lock.Lock()
	defer ns.lock.Unlock()
	if oldEntry, ok := ns.nsEntrys[key]; ok {
		if oldEntry.isExpired() == false && oldEntry.trustLevel >= trustLevel {
			return
		}
	}
	ns.nsEntrys[name.Hash(false)] = e
}

func (ns *NameServerManager) getNameServer(name *g53.Name) *NameServerEntry {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	hash := name.Hash(false)
	if e, ok := ns.nsEntrys[hash]; ok && e.name.Equals(name) {
		return e
	} else {
		return nil
	}
}

func (ns *NameServerManager) updateRtt(nameServer *NameServer, rtt time.Duration) error {
	ns.lock.Lock()
	defer ns.lock.Unlock()

	if e, ok := ns.nsEntrys[nameServer.name.Hash(false)]; ok && e.name.Equals(nameServer.name) {
		e.updateRtt(nameServer, rtt)
		return nil
	} else {
		return errUnknownNameServer
	}
}

func (ns *NameServerManager) deleteNameServers(names []*g53.Name) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	for _, name := range names {
		delete(ns.nsEntrys, name.Hash(false))
	}
}

type ServerByRtt []*NameServer

func (s ServerByRtt) Len() int           { return len(s) }
func (s ServerByRtt) Less(i, j int) bool { return s[i].rtt < s[j].rtt }
func (s ServerByRtt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func cloneNameServers(src []*NameServer) []*NameServer {
	c := len(src)
	if c == 0 {
		panic("clone empty name servers")
	}
	dst := make([]*NameServer, c)
	copy(dst, src)
	return dst
}
