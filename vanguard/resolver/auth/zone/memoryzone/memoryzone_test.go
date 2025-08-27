package memoryzone

import (
	"fmt"
	"testing"

	ut "cement/unittest"
	"g53"
	zn "vanguard/resolver/auth/zone"
)

var zoneData = []string{
	"example.org. 300 IN SOA xxx.net. ns.example.org. 100 1800 900 604800 86400",
	"example.org. 300 IN NS ns.example.org.",
	"example.org. 300 IN A 192.0.2.1",
	"ns.example.org. 300 IN A 192.0.2.2",
	"ns.example.org. 300 IN AAAA 2001:db8::2",
	"cname.example.org. 300 IN CNAME canonical.example.org",
	"dname.example.org. 300 IN NS ns.dname.example.org.",
	"child.example.org. 300 IN NS ns.child.example.org.",
	"ns.child.example.org. 300 IN A 192.0.2.153",
	"grand.child.example.org. 300 IN NS ns.grand.child.example.org.",
	"ns.grand.child.example.org. 300 IN AAAA 2001:db8::253",
	"foo.wild.example.org. 300 IN A 192.0.2.3",
	"wild.*.foo.example.org. 300 IN A 192.0.2.1",
	"wild.*.foo.*.bar.example.org. 300 IN A 192.0.2.1",
	"bar.foo.wild.example.org. 300 IN A 192.0.2.2",
	"baz.foo.wild.example.org. 300 IN A 192.0.2.3",
}

func createZone(name string, data []string) *MemoryZone {
	zone := newMemoryZone(g53.NameFromStringUnsafe(name))
	for _, rr := range data {
		rrset, err := g53.RRsetFromString(rr)
		if err != nil {
			panic(fmt.Sprintf("rr %s isn't valid %vs", rr, err.Error()))
		}
		zone.addRRset(rrset)
	}
	return zone
}

func TestAddRRset(t *testing.T) {
	zone := createZone("example.org.", zoneData)
	outRR, _ := g53.RRsetFromString("example.com. 300 IN A 192.0.2.10")
	ut.Equal(t, zone.addRRset(outRR), zn.ErrOutOfZone)

	// Now put all the data we have there. It should throw nothing
	ns, _ := g53.RRsetFromString("example.org. 300 IN NS ns.example.org.")
	ut.Equal(t, zone.addRRset(ns), g53.ErrDuplicateRdata)
	nsGlue, _ := g53.RRsetFromString("ns.example.org. 300 IN A 192.0.2.2")
	ut.Equal(t, zone.addRRset(nsGlue), g53.ErrDuplicateRdata)

	nsGlue, _ = g53.RRsetFromString("ns.example.org. 300 IN A 192.1.2.2")
	ut.Assert(t, zone.addRRset(nsGlue) == nil, "new glue should ok")

	a, _ := g53.RRsetFromString("cname.example.org. 300 IN A 192.0.2.3")
	ut.Equal(t, zone.addRRset(a), zn.ErrCNAMECoExistsWithOtherRR)

	//add new cname record will overwrite old one
	cname, _ := g53.RRsetFromString("cname.example.org. 300 IN CNAME www.baidu.com")
	ut.Assert(t, zone.addRRset(cname) == nil, "")
	result := zone.find(g53.NameFromStringUnsafe("cname.example.org"), g53.RR_CNAME, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.RRCount(), 1)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "www.baidu.com.")

	cname, _ = g53.RRsetFromString("foo.wild.example.org. 300 IN CNAME www.baidu.com")
	ut.Equal(t, zone.addRRset(cname), zn.ErrCNAMECoExistsWithOtherRR)
	newSOA, _ := g53.RRsetFromString("example.org. 300 IN SOA xxx.net. ns.example.org. 88 1800 900 604800 86400")
	ut.Assert(t, zone.addRRset(newSOA) == nil, "")
	result = zone.find(g53.NameFromStringUnsafe("example.org"), g53.RR_SOA, zn.DefaultFind).GetResult()
	ut.Equal(t, result.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(100))

	newSOA, _ = g53.RRsetFromString("example.org. 300 IN SOA xxx.net. ns.example.org. 200 1800 900 604800 86400")
	ut.Assert(t, zone.addRRset(newSOA) == nil, "")
	result = zone.find(g53.NameFromStringUnsafe("example.org"), g53.RR_SOA, zn.DefaultFind).GetResult()
	ut.Equal(t, result.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(200))

	a, _ = g53.RRsetFromString("bar.foo.wild.example.org. 1300 IN A 192.0.2.2")
	ut.Assert(t, zone.addRRset(a) == nil, "update ttl should succeed but get %v", zone.addRRset(a))
	result = zone.find(g53.NameFromStringUnsafe("bar.foo.wild.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.RRset.RRCount(), 1)
}

func TestCNAMEFind(t *testing.T) {
	zone := createZone("example.org.", zoneData)
	result := zone.find(g53.NameFromStringUnsafe("cname.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRCname)
	ut.Equal(t, result.RRset.Type, g53.RR_CNAME)

	result = zone.find(g53.NameFromStringUnsafe("cname.example.org."), g53.RR_CNAME, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "canonical.example.org.")

	cname, _ := g53.RRsetFromString("cname.child.example.org. 300 IN CNAME www.knet.cn")
	zone.addRRset(cname)
	result = zone.find(g53.NameFromStringUnsafe("cname.child.example.org."), g53.RR_AAAA, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRCname)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "www.knet.cn.")
}

func TestDelegation(t *testing.T) {
	zone := createZone("example.org.", zoneData)

	result := zone.find(g53.NameFromStringUnsafe("www.child.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("child.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("child.example.org."), g53.RR_NS, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("example.org."), g53.RR_NS, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("www.grand.child.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")
}

func TestFindGlue(t *testing.T) {
	zone := createZone("example.org.", zoneData)

	result := zone.find(g53.NameFromStringUnsafe("ns.child.example.org"), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("ns.child.example.org"), g53.RR_A, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "192.0.2.153")

	result = zone.find(g53.NameFromStringUnsafe("ns.child.example.org."), g53.RR_AAAA, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("www.child.example.org."), g53.RR_A, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("ns.grand.child.example.org."), g53.RR_AAAA, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "2001:db8::253")

	result = zone.find(g53.NameFromStringUnsafe("www.grand.child.example.org."), g53.RR_TXT, zn.GlueOkFind).GetResult()
	ut.Equal(t, result.Type, zn.FRDelegation)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.child.example.org.")

}

func TestGenralFind(t *testing.T) {
	zone := createZone("example.org.", zoneData)

	result := zone.find(g53.NameFromStringUnsafe("example.org."), g53.RR_NS, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "ns.example.org.")

	result = zone.find(g53.NameFromStringUnsafe("ns.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.Rdatas[0].String(), "192.0.2.2")

	result = zone.find(g53.NameFromStringUnsafe("example.org."), g53.RR_AAAA, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("ns.example.org."), g53.RR_NS, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("nothere.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)

	result = zone.find(g53.NameFromStringUnsafe("example.net."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)
}

func TestEmptyNode(t *testing.T) {
	/*
	   example.org
	        |
	       baz (empty; easy case)
	     /  |  \
	   bar  |  x.foo ('foo' part is empty; a bit trickier)
	       bbb
	      /
	    aaa
	*/
	zone := createZone("example.org.", []string{
		"example.org. 300 IN A 192.0.2.1",
		"bar.example.org. 300 IN A 192.0.2.1",
		"x.foo.example.org. 300 IN A 192.0.2.1",
		"aaa.baz.example.org. 300 IN A 192.0.2.1",
		"bbb.baz.example.org. 300 IN A 192.0.2.1",
	})

	result := zone.find(g53.NameFromStringUnsafe("baz.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("foo.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)

}

func TestDelete(t *testing.T) {
	zone := createZone("example.org.", zoneData)

	ns, _ := g53.RRsetFromString("example.com. 300 IN NS ns.example.org.")
	_, err := zone.deleteRRset(ns)
	ut.Equal(t, err, zn.ErrOutOfZone)

	nsGlue, _ := g53.RRsetFromString("ns.example.org. 300 IN A 192.0.2.2")
	_, err = zone.deleteRRset(nsGlue)
	ut.Assert(t, err == nil, "")

	result := zone.find(g53.NameFromStringUnsafe("ns.example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	txt, _ := g53.RRsetFromString("baz.foo.wild.example.org. 300 IN TXT \"just for test\"")
	ut.Assert(t, zone.addRRset(txt) == nil, "")

	a, _ := g53.RRsetFromString("baz.foo.wild.example.org. 300 IN A 2.2.2.2")
	ut.Assert(t, zone.addRRset(a) == nil, "")

	//delete unknown rr, just ignore now
	/*a, _ = g53.RRsetFromString("baz.foo.wild.example.org. 300 IN A 192.0.0.30")
	_, err = zone.deleteRr(a)
	ut.Assert(t, err != nil, "delete not exist rr should fail")
	*/

	a, _ = g53.RRsetFromString("baz.foo.wild.example.org. 300 IN A 192.0.2.3")
	_, err = zone.deleteRr(a)
	ut.Assert(t, err == nil, "")

	result = zone.find(g53.NameFromStringUnsafe("baz.foo.wild.example.org"), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.RRCount(), 1)

	a.Rdatas = nil
	_, err = zone.deleteRRset(a)
	ut.Assert(t, err == nil, "")
	result = zone.find(g53.NameFromStringUnsafe("baz.foo.wild.example.org"), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	_, err = zone.deleteRRset(txt)
	ut.Assert(t, err == nil, "")
	result = zone.find(g53.NameFromStringUnsafe("baz.foo.wild.example.org"), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)

	soa, _ := g53.RRsetFromString("example.org. 300 IN SOA xxx.net. ns.example.org. 100 1800 900 604800 86400")
	_, err = zone.deleteRRset(soa)
	ut.Equal(t, err, zn.ErrShortOfSOA)

	ns, _ = g53.RRsetFromString("example.org. 300 IN NS ns.example.org.")
	_, err = zone.deleteRRset(ns)
	ut.Equal(t, err, zn.ErrShortOfNS)

	oldNode, err := zone.deleteDomain(g53.NameFromStringUnsafe("example.org."))
	ut.Equal(t, err, nil)
	ut.Equal(t, oldNode.IsEmpty(), false)
	result = zone.find(g53.NameFromStringUnsafe("example.org."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)
}

func TestWildcard(t *testing.T) {
	var zoneData = []string{
		"*.example.              3600     IN TXT   \"this is a wildcard\"",
		"*.example.               3600     IN MX    10 host1.example.",
		"*.c.example.               3600     IN A 2.2.2.2",
		"sub.*.example.           3600     IN TXT   \"this is not a wildcard\"",
		"host1.example.           3600     IN A     192.0.2.1",
		"_ssh._tcp.host1.example. 3600     IN SRV 10 60 5060 bigbox.example.com.",
		"_ssh._tcp.host2.example. 3600     IN SRV 10 60 5060 b.c.example.",
		"subdel.example.          3600     IN NS    ns.example.com.",
		"subdel.example.          3600     IN NS    ns.example.net.",
	}
	zone := createZone("example.", zoneData)

	result := zone.find(g53.NameFromStringUnsafe("host3.example."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	result = zone.find(g53.NameFromStringUnsafe("host3.example."), g53.RR_MX, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.RRCount(), 1)

	result = zone.find(g53.NameFromStringUnsafe("foo.bar.example."), g53.RR_TXT, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
	ut.Equal(t, result.RRset.RRCount(), 1)

	result = zone.find(g53.NameFromStringUnsafe("ghost.*.example."), g53.RR_TXT, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)

	result = zone.find(g53.NameFromStringUnsafe("b.c.example."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRSuccess)
}

func TestGlueFind(t *testing.T) {
	var zoneData = []string{
		"example.org. 300 IN SOA xxx.net. ns.example.org. 100 1800 900 604800 86400",
		"example.org. 300 IN NS ns.example.org.",
		"ns.example.org. 300 IN A 192.0.2.2",
		"_service._proto.example.org. 100 IN SRV 1 2 3 glue.example.org.",
		"glue.example.org. 100 IN A 1.1.1.4",
	}
	zone := createZone("example.org.", zoneData)
	ctx := zone.find(g53.NameFromStringUnsafe("_service._proto.example.org."), g53.RR_SRV, zn.DefaultFind)
	result := ctx.GetResult()
	ut.Equal(t, result.RRset.Type, g53.RR_SRV)

	glue := ctx.GetAdditional()
	ut.Equal(t, len(glue), 1)
	ut.Equal(t, glue[0].Type, g53.RR_A)
	ut.Equal(t, glue[0].Rdatas[0].String(), "1.1.1.4")
}
