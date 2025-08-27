package recursor

import (
	"testing"

	ut "cement/unittest"
	"g53"
	"g53/util"
	"vanguard/logger"
)

/*
isc.org. IN NS

;; ANSWER SECTION:
isc.org.	7200	IN	NS	sfba.sns-pb.isc.org.
isc.org.	7200	IN	NS	ord.sns-pb.isc.org.
isc.org.	7200	IN	NS	ams.sns-pb.isc.org.
isc.org.	7200	IN	NS	ns.isc.afilias-nst.info.

;; ADDITIONAL SECTION:
ams.sns-pb.isc.org.	7200	IN	A	199.6.1.30
ams.sns-pb.isc.org.	7200	IN	AAAA	2001:500:60::30
ord.sns-pb.isc.org.	7200	IN	A	199.6.0.30
ord.sns-pb.isc.org.	7200	IN	AAAA	2001:500:71::30
sfba.sns-pb.isc.org.	7200	IN	A	149.20.64.3
sfba.sns-pb.isc.org.	7200	IN	AAAA	2001:4f8:0:2::19
*/

func buildISCORGNSMessage() *g53.Message {
	iscNsResponseRaw := "04b08500000100040000000703697363036f72670000020001c00c0002000100001c20000e047366626106736e732d7062c00cc00c0002000100001c200006036f7264c02ac00c0002000100001c20000603616d73c02ac00c0002000100001c200019026e73036973630b6166696c6961732d6e737404696e666f00c0510001000100001c200004c706011ec051001c000100001c20001020010500006000000000000000000030c03f0001000100001c200004c706001ec03f001c000100001c20001020010500007100000000000000000030c0250001000100001c20000495144003c025001c000100001c200010200104f80000000200000000000000190000291000000000000000"
	wire, _ := util.HexStrToBytes(iscNsResponseRaw)
	buf := util.NewInputBuffer(wire)
	nm, _ := g53.MessageFromWire(buf)
	return nm
}

func TestNSASCacheSelectNameServer(t *testing.T) {
	cache := NewNsasCache(10)
	missing, known := cache.AddZoneNameServer(g53.NameFromStringUnsafe("isc.org."), buildISCORGNSMessage())
	ut.Equal(t, len(missing), 1)
	ut.Assert(t, missing[0].Equals(g53.NameFromStringUnsafe("ns.isc.afilias-nst.info.")), "")
	ut.Equal(t, len(known), 3)
	ut.Equal(t, cache.zoneCount(), 1)
	cache.EnforceMemoryLimit()
	ut.Equal(t, cache.zoneCount(), 1)

	nameServers := cache.SelectNameServers(g53.NameFromStringUnsafe("xxx.isc.org."))
	ut.Equal(t, len(nameServers), 3)
	nameServers = cache.SelectNameServers(g53.NameFromStringUnsafe("xxx.isc.org."))
	ut.Equal(t, len(nameServers), 3)
	nameServers = cache.SelectNameServers(g53.NameFromStringUnsafe("org."))
	ut.Equal(t, len(nameServers), 0)

	for _, ns := range []string{"ord.sns-pb.isc.org.", "ams.sns-pb.isc.org.", "sfba.sns-pb.isc.org."} {
		nameServer := cache.nameServers.getNameServer(g53.NameFromStringUnsafe(ns))
		//only v4 address is added
		ut.Equal(t, len(nameServer.addrEntrys), 1)
	}
}

func buildHeader(id uint16, setFlag []g53.FlagField, counts []uint16, opcode g53.Opcode, rcode g53.Rcode) g53.Header {
	h := g53.Header{
		Id:      id,
		Opcode:  opcode,
		Rcode:   rcode,
		QDCount: counts[0],
		ANCount: counts[1],
		NSCount: counts[2],
		ARCount: counts[3],
	}

	for _, f := range setFlag {
		h.SetFlag(f, true)
	}

	return h
}

func buildFackNSResponse(zone *g53.Name) *g53.Message {
	nsName1, _ := g53.NameFromStringUnsafe("ns1").Concat(zone)
	nsName2, _ := g53.NameFromStringUnsafe("ns2").Concat(zone)

	ns1, _ := g53.NSFromString(nsName1.String(false))
	ns2, _ := g53.NSFromString(nsName2.String(false))
	var answer g53.Section
	answer = append(answer, &g53.RRset{
		Name:   zone,
		Type:   g53.RR_NS,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{ns1, ns2},
	})

	ra1, _ := g53.AFromString("1.1.1.1")
	ra2, _ := g53.AFromString("2.2.2.2")
	ra3, _ := g53.AFromString("3.3.3.3")
	ra4, _ := g53.AFromString("4.4.4.4")
	var additional g53.Section
	additional = append(additional, &g53.RRset{
		Name:   nsName1,
		Type:   g53.RR_A,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{ra1, ra2},
	}, &g53.RRset{
		Name:   nsName2,
		Type:   g53.RR_A,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{ra3, ra4},
	})

	return &g53.Message{
		Header: buildHeader(uint16(1200), []g53.FlagField{g53.FLAG_QR, g53.FLAG_AA, g53.FLAG_RD}, []uint16{1, 2, 0, 4}, g53.OP_QUERY, g53.R_NOERROR),
		Question: &g53.Question{
			Name:  zone,
			Type:  g53.RR_NS,
			Class: g53.CLASS_IN,
		},
		Sections: [...]g53.Section{answer, nil, additional},
		Edns: &g53.EDNS{
			UdpSize:     uint16(4096),
			DnssecAware: false,
		},
	}
}

func TestNSASCacheMemoryEnforce(t *testing.T) {
	logger.UseDefaultLogger("error")
	cache := NewNsasCache(4)
	for _, name := range []string{"knet.cn.", "knet.com", "com.", "cn.", "org."} {
		zone := g53.NameFromStringUnsafe(name)
		missing, known := cache.AddZoneNameServer(zone, buildFackNSResponse(zone))
		ut.Equal(t, len(missing), 0)
		ut.Equal(t, len(known), 2)
	}
	ut.Equal(t, cache.zoneCount(), 5)
	nameServer := cache.nameServers.getNameServer(g53.NameFromStringUnsafe("ns1.knet.cn"))
	ut.Equal(t, len(nameServer.addrEntrys), 2)
	cache.EnforceMemoryLimit()
	ut.Equal(t, cache.zoneCount(), 4)
	nameServers := cache.SelectNameServers(g53.NameFromStringUnsafe("a.knet.cn"))
	ut.Equal(t, len(nameServers), 2)
	ut.Assert(t, nameServers[0].zone.Equals(g53.NameFromStringUnsafe("cn.")), "")
	ut.Assert(t, nameServers[1].zone.Equals(g53.NameFromStringUnsafe("cn.")), "")
	nameServer = cache.nameServers.getNameServer(g53.NameFromStringUnsafe("ns1.knet.cn"))
	ut.Assert(t, nameServer == nil, "")

	knet_cn := g53.NameFromStringUnsafe("knet.cn.")
	cache.AddZoneNameServer(knet_cn, buildFackNSResponse(knet_cn))
	cache.SelectNameServers(g53.NameFromStringUnsafe("knet.com"))
	cache.EnforceMemoryLimit()
	ut.Equal(t, cache.zoneCount(), 4)
	nameServers = cache.SelectNameServers(g53.NameFromStringUnsafe("a.com."))
	ut.Equal(t, len(nameServers), 0)
	nameServer = cache.nameServers.getNameServer(g53.NameFromStringUnsafe("ns1.com."))
	ut.Assert(t, nameServer == nil, "")
	nameServer = cache.nameServers.getNameServer(g53.NameFromStringUnsafe("ns2.com."))
	ut.Assert(t, nameServer == nil, "")
}
