package recursor

import (
	"container/list"
	"sync"
	"time"

	"cement/domaintree"
	"g53"
	"vanguard/logger"
)

const (
	DefaultMaxCacheSize    = 4096
	EmptyNodeWaterMark     = 40 //40%
	EmptyNodeCheckInterval = time.Second
	ExpiredZoneCheckBatch  = 100
)

type NsasCache struct {
	zones        *domaintree.DomainTree
	visitedZone  *list.List
	zonesLock    sync.Mutex
	nameServers  *NameServerManager
	maxCacheSize int
}

func NewNsasCache(maxCacheSize int) *NsasCache {
	if maxCacheSize <= 0 {
		maxCacheSize = DefaultMaxCacheSize
	}

	cache := &NsasCache{
		zones:        domaintree.NewDomainTree(),
		visitedZone:  list.New(),
		nameServers:  newNameServerManager(),
		maxCacheSize: maxCacheSize,
	}
	return cache
}

func (nc *NsasCache) AddZoneNameServer(zone *g53.Name, msg *g53.Message) ([]*g53.Name, []*g53.Name) {
	nsRRset, glues, err := getAuthAndGlues(zone, msg)
	if err != nil {
		return nil, nil
	}

	for _, glue := range glues {
		if glue.Name.IsSubDomain(zone) == false {
			nc.addNameServer(glue, OutOfZone)
		} else {
			nc.addNameServer(glue, FromAuth)
		}
	}

	trustLevel := FromReferal
	if msg.Header.GetFlag(g53.FLAG_AA) {
		trustLevel = FromAuth
	}
	return nc.addZone(nsRRset, trustLevel)
}

func (nc *NsasCache) zoneCount() int {
	return nc.visitedZone.Len()
}

func (nc *NsasCache) addZone(nsRRset *g53.RRset, trustLevel TrustLevel) ([]*g53.Name, []*g53.Name) {
	zone := nsRRset.Name
	serverNames := []*g53.Name{}
	missingServerNames := []*g53.Name{}
	knownServerNames := []*g53.Name{}
	for _, nsRdata := range nsRRset.Rdatas {
		serverName := nsRdata.(*g53.NS).Name
		serverNames = append(serverNames, serverName)
		e := nc.nameServers.getNameServer(serverName)
		if e == nil || e.isExpired() {
			missingServerNames = append(missingServerNames, serverName)
		} else if e.trustLevel == OutOfZone {
			//we will use the server this time, but probe the server in backend thread
			missingServerNames = append(missingServerNames, serverName)
			knownServerNames = append(knownServerNames, serverName)
		} else {
			knownServerNames = append(knownServerNames, serverName)
		}
	}

	nc.zonesLock.Lock()
	defer nc.zonesLock.Unlock()
	_, node, searchResult := nc.zones.Search(zone)
	if searchResult == domaintree.ExactMatch {
		elem := node.(*list.Element)
		oldEntry := elem.Value.(*ZoneEntry)
		if oldEntry.isExpired() == false && oldEntry.trustLevel > trustLevel {
			nc.visitedZone.MoveToFront(elem)
			return nil, nil
		}
	}

	zoneEntry := newZoneEntry(zone, serverNames, time.Duration(nsRRset.Ttl)*time.Second, trustLevel)
	elem := nc.visitedZone.PushFront(zoneEntry)
	nc.zones.Insert(zone, elem)
	return missingServerNames, knownServerNames
}

func (nc *NsasCache) SelectNameServers(zone *g53.Name) []*NameServer {
	nc.zonesLock.Lock()
	defer nc.zonesLock.Unlock()
	return nc.selectNameServers(zone)
}

func (nc *NsasCache) selectNameServers(zone *g53.Name) []*NameServer {
	_, node, searchResult := nc.zones.Search(zone)
	if searchResult == domaintree.NotFound {
		return nil
	}

	elem := node.(*list.Element)
	e := elem.Value.(*ZoneEntry)
	if e.isExpired() {
		nc.removeZone(elem)
		return nc.selectNameServers(zone)
	} else {
		servers := e.selectNameServer(nc.nameServers)
		if len(servers) == 0 {
			nc.removeZone(elem)
			return nc.selectNameServers(zone)
		} else {
			nc.visitedZone.MoveToFront(elem)
			return servers
		}
	}
}

func (nc *NsasCache) removeZone(elem *list.Element) {
	e := elem.Value.(*ZoneEntry)
	nc.zones.Delete(e.zone)
	nc.visitedZone.Remove(elem)
	nc.nameServers.deleteNameServers(elem.Value.(*ZoneEntry).nameServers)
}

func (nc *NsasCache) UpdateRtt(server *NameServer, rtt time.Duration) error {
	return nc.nameServers.updateRtt(server, rtt)
}

func (nc *NsasCache) addNameServer(glue *g53.RRset, trustLevel TrustLevel) {
	addrs := []string{}
	for _, rdata := range glue.Rdatas {
		addrs = append(addrs, rdata.String()+":53")
	}
	nc.nameServers.addNameServer(glue.Name, time.Duration(glue.Ttl)*time.Second, addrs, trustLevel)
}

func (nc *NsasCache) EnforceMemoryLimit() {
	nc.zonesLock.Lock()
	defer nc.zonesLock.Unlock()
	zoneCount := nc.zoneCount()
	if zoneCount <= nc.maxCacheSize {
		return
	}

	logger.GetLogger().Info("current zone count %d", zoneCount)
	elem := nc.visitedZone.Back()
	expiredZone := 0
	for i := 0; i < ExpiredZoneCheckBatch && elem != nil; i++ {
		if elem.Value.(*ZoneEntry).isExpired() {
			prev := elem.Prev()
			nc.removeZone(elem)
			expiredZone += 1
			elem = prev
		} else {
			elem = elem.Prev()
		}
	}
	logger.GetLogger().Info("remove expired zone %d", expiredZone)

	zonesToRemove := zoneCount - nc.maxCacheSize
	logger.GetLogger().Info("zone count %d, and list len %d", zoneCount, nc.zoneCount())
	if zonesToRemove > 0 {
		logger.GetLogger().Info("remove last recent used zone %d", zonesToRemove)
		for i := 0; i < zonesToRemove; i++ {
			elem := nc.visitedZone.Back()
			nc.removeZone(elem)
		}
	}
}
