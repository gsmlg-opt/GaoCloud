package sortlist

import (
	"errors"
	"net"
	"sort"
	"sync"

	"cement/netradix"
	"g53"
	"vanguard/config"
	"vanguard/httpcmd"
)

const (
	//gDefaultSortList = "default"
	maxOrderCnt = 65535
)

var (
	ErrUnknownSortlist = errors.New("unknown sort list")
)

type RRsetSorter interface {
	ReloadConfig(*config.VanguardConf)
	Sort(string, net.IP, *g53.RRset) *g53.RRset
}

type rrsetSortWrapper struct {
	priorities map[g53.Rdata]int
	rrset      *g53.RRset
}

func (s *rrsetSortWrapper) Len() int {
	return len(s.rrset.Rdatas)
}

func (s *rrsetSortWrapper) Swap(i, j int) {
	s.rrset.Rdatas[i], s.rrset.Rdatas[j] = s.rrset.Rdatas[j], s.rrset.Rdatas[i]
}

func (s *rrsetSortWrapper) Less(i, j int) bool {
	return s.priorities[s.rrset.Rdatas[i]] < s.priorities[s.rrset.Rdatas[j]]
}

type UserAddrBasedSorter struct {
	viewSortLists map[string]*netradix.NetRadixTree
	lock          sync.RWMutex
}

func newUserAddrBasedSorter() RRsetSorter {
	sorter := &UserAddrBasedSorter{}
	httpcmd.RegisterHandler(sorter, []httpcmd.Command{&AddSortList{}, &DeleteSortList{}, &UpdateSortList{}})
	return sorter
}

func (sorter *UserAddrBasedSorter) ReloadConfig(conf *config.VanguardConf) {
	sorter.viewSortLists = make(map[string]*netradix.NetRadixTree)
	for _, c := range conf.SortList {
		if err := sorter.addSorter(c.View, c.SourceIp, c.PreferredIps); err != nil {
			panic("sortlist load config failed:" + err.Error())
		}
	}
}

func (sorter *UserAddrBasedSorter) Sort(view string, sourceIP net.IP, rrset *g53.RRset) *g53.RRset {
	if len(rrset.Rdatas) < 2 || (rrset.Type != g53.RR_A && rrset.Type != g53.RR_AAAA) {
		return rrset
	}

	sorter.lock.RLock()
	sourceTree, ok := sorter.viewSortLists[view]
	sorter.lock.RUnlock()
	if ok == false {
		return rrset
	}

	if tree, found := sourceTree.SearchBest(sourceIP); found {
		rrOrder := make(map[g53.Rdata]int)
		orderTree := tree.(*netradix.NetRadixTree)
		var host net.IP
		for _, rdata := range rrset.Rdatas {
			if rrset.Type == g53.RR_A {
				host = rdata.(*g53.A).Host
			} else {
				host = rdata.(*g53.AAAA).Host
			}

			if order, found := orderTree.SearchBest(host); found {
				rrOrder[rdata] = order.(int)
			} else {
				rrOrder[rdata] = maxOrderCnt
			}
		}

		rrsetClone := rrset.Clone()
		sort.Sort(&rrsetSortWrapper{
			priorities: rrOrder,
			rrset:      rrsetClone,
		})
		return rrsetClone
	}

	return rrset
}

func (sorter *UserAddrBasedSorter) addSorter(view, sourceIp string, preferedIps []string) error {
	orderTree := newOrderTree(preferedIps)
	sorter.lock.Lock()
	defer sorter.lock.Unlock()
	sourceTree, ok := sorter.viewSortLists[view]
	if ok == false {
		sourceTree = netradix.NewNetRadixTree()
		sorter.viewSortLists[view] = sourceTree
	}
	return sourceTree.Add(sourceIp, orderTree)
}

func (sorter *UserAddrBasedSorter) deleteSorter(view, sourceIp string) error {
	sorter.lock.Lock()
	defer sorter.lock.Unlock()
	if tree, ok := sorter.viewSortLists[view]; ok {
		return tree.Delete(sourceIp)
	} else {
		return ErrUnknownSortlist
	}
}

func (sorter *UserAddrBasedSorter) updateSorter(view, sourceIp string, preferedIps []string) error {
	orderTree := newOrderTree(preferedIps)
	sorter.lock.Lock()
	defer sorter.lock.Unlock()
	sourceTree, ok := sorter.viewSortLists[view]
	if ok == false {
		return ErrUnknownSortlist
	}

	if err := sourceTree.Delete(sourceIp); err != nil {
		return err
	}
	return sourceTree.Add(sourceIp, orderTree)
}

func newOrderTree(preferedIps []string) *netradix.NetRadixTree {
	orderTree := netradix.NewNetRadixTree()
	for i, ip := range preferedIps {
		orderTree.Add(ip, i)
	}
	return orderTree
}
