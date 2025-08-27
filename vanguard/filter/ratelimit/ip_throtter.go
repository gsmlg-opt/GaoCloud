package ratelimit

import (
	"cement/netradix"
	"net"
	"sync"

	"vanguard/config"
	"vanguard/httpcmd"
)

const maxIPAccessRecordCount = 5000

type throttlerEntry struct {
	records  *IPAccessRecordStore
	networks *netradix.NetRadixTree
}

type IPThrottler struct {
	entry *throttlerEntry
	lock  sync.Mutex
}

func NewIPThrotter(conf *config.VanguardConf) *IPThrottler {
	t := &IPThrottler{
		entry: &throttlerEntry{},
	}
	t.ReloadConfig(conf)
	httpcmd.RegisterHandler(t, []httpcmd.Command{&AddIpRateLimit{}, &DeleteIpRateLimit{}, &UpdateIpRateLimit{}})
	return t
}

func (t *IPThrottler) ReloadConfig(conf *config.VanguardConf) {
	networks := netradix.NewNetRadixTree()
	records := newIPAccessRecordStore(maxIPAccessRecordCount)
	for _, limit := range conf.Filter.NetworkLimit {
		if err := networks.Add(limit.Network, limit.Limit); err != nil {
			panic("load ip " + limit.Network + " failed: " + err.Error())
		}
	}

	t.entry.networks = networks
	t.entry.records = records
}

func (t *IPThrottler) IsIPAllowed(ip net.IP) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.entry.isIpAllowed(ip)
}

func (t *throttlerEntry) isIpAllowed(ip net.IP) bool {
	limit, found := t.networks.SearchBest(ip)
	if found == false {
		return true
	}

	record, ok := t.records.Get(ip)
	if ok == false {
		t.records.Add(newIPAccessRecord(ip))
		return true
	}
	return record.IsAccessAllowed(limit.(uint32))
}
