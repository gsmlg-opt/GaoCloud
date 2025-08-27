package sortlist

import (
	"net"

	"cement/netradix"
	ut "cement/unittest"
	"g53"
	"testing"
)

func createRRset(ips []string, typ g53.RRType) *g53.RRset {
	n, _ := g53.NameFromString("test.example.com.")
	rrset := &g53.RRset{
		Name:   n,
		Type:   typ,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{},
	}
	for _, ip := range ips {
		ra, _ := g53.RdataFromString(typ, ip)
		rrset.Rdatas = append(rrset.Rdatas, ra)
	}
	return rrset
}

func sortIPs(rrsetSorter RRsetSorter, viewName, clientAddr string, originIPs []string) []string {
	addr, _ := net.ResolveUDPAddr("udp", clientAddr+":")
	rrset := createRRset(originIPs, g53.RR_A)
	sortedRRset := rrsetSorter.Sort(viewName, addr.IP, rrset)

	var sortedIPs []string
	for _, rdata := range sortedRRset.Rdatas {
		sortedIPs = append(sortedIPs, rdata.String())
	}
	return sortedIPs
}

func TestIpV4(t *testing.T) {
	sorter := &UserAddrBasedSorter{
		viewSortLists: make(map[string]*netradix.NetRadixTree),
	}

	sorter.addSorter("v1", "1.1.1.0/24", []string{"2.2.0.0/16", "3.3.0.0/16", "4.4.0.0/16"})
	originIPs := []string{"1.1.1.1", "4.4.4.4", "3.3.3.3", "2.2.2.2"}
	expectIPs := []string{"2.2.2.2", "3.3.3.3", "4.4.4.4", "1.1.1.1"}
	rrset := createRRset(originIPs, g53.RR_A)
	addr, _ := net.ResolveUDPAddr("udp", "1.1.1.3:")
	newRRset := sorter.Sort("v1", addr.IP, rrset)
	for i, rdata := range newRRset.Rdatas {
		ut.Equal(t, rdata.String(), expectIPs[i])
	}

	sorter.updateSorter("v1", "1.1.1.0/24", []string{"3.3.0.0/16", "2.2.0.0/16", "4.4.0.0/16"})
	expectIPs = []string{"3.3.3.3", "2.2.2.2", "4.4.4.4", "1.1.1.1"}
	newRRset = sorter.Sort("v1", addr.IP, rrset)
	for i, rdata := range newRRset.Rdatas {
		ut.Equal(t, rdata.String(), expectIPs[i])
	}

	sorter.deleteSorter("v1", "1.1.1.0/24")
	newRRset = sorter.Sort("v1", addr.IP, rrset)
	for i, rdata := range newRRset.Rdatas {
		ut.Equal(t, rdata.String(), originIPs[i])
	}
}

func TestIpV6(t *testing.T) {
	sorter := &UserAddrBasedSorter{
		viewSortLists: make(map[string]*netradix.NetRadixTree),
	}

	sorter.addSorter("v6", "1.1.1.0/24", []string{"2:2:2:2:2:2:2:2", "3:3:3:3:3:3:3:3", "1:1:1:1:1:1:1:1"})
	originIPs := []string{"1:1:1:1:1:1:1:1", "2:2:2:2:2:2:2:2", "3:3:3:3:3:3:3:3"}
	expectIPs := []string{"2:2:2:2:2:2:2:2", "3:3:3:3:3:3:3:3", "1:1:1:1:1:1:1:1"}
	rrset := createRRset(originIPs, g53.RR_AAAA)
	addr, _ := net.ResolveUDPAddr("udp", "1.1.1.3:")
	newRRset := sorter.Sort("v6", addr.IP, rrset)
	for i, rdata := range newRRset.Rdatas {
		ut.Equal(t, rdata.String(), expectIPs[i])
	}
}
