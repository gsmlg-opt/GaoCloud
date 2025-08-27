package ratelimit

import (
	"sync"

	"cement/domaintree"
	"g53"
	"vanguard/config"
	"vanguard/httpcmd"
)

type NameThrottler struct {
	viewRecords map[string]*domaintree.DomainTree
	lock        sync.Mutex
}

func NewNameThrotter(conf *config.VanguardConf) *NameThrottler {
	t := &NameThrottler{}
	t.ReloadConfig(conf)
	httpcmd.RegisterHandler(t, []httpcmd.Command{&AddNameRateLimit{}, &DeleteNameRateLimit{}, &UpdateNameRateLimit{}})
	return t
}

func (t *NameThrottler) ReloadConfig(conf *config.VanguardConf) {
	records := make(map[string]*domaintree.DomainTree)
	for _, limitsForView := range conf.Filter.DomainNameLimit {
		tree := domaintree.NewDomainTree()
		records[limitsForView.View] = tree
		for _, limit := range limitsForView.DomainNameLimit {
			if err := doAddNameRateLimit(tree, limit.Name, limit.Limit); err != nil {
				panic("load name rrls " + limit.Name + " failed: " + err.Error())
			}
		}
	}
	t.viewRecords = records
}

func (t *NameThrottler) IsNameAllowed(view string, name *g53.Name) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	records, ok := t.viewRecords[view]
	if ok == false {
		return true
	}

	parents, match := records.SearchParents(name)
	if match == domaintree.ExactMatch {
		return parents.Top().Data().(*NameAccessRecord).IsAccessAllowed()
	}

	if match == domaintree.ClosestEncloser {
		for parents.IsEmpty() == false {
			data := parents.Top().Data()
			if data != nil {
				parent_record := data.(*NameAccessRecord)
				if parent_record.match_type == zoneMatch {
					return parent_record.IsAccessAllowed()
				}
			}
			parents.Pop()
		}
	}
	return true
}
