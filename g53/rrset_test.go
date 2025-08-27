package g53

import (
	"fmt"
	"net"
	"testing"

	"g53/util"
)

func matchRRsetRaw(t *testing.T, rawData string, rs *RRset) {
	wire, _ := util.HexStrToBytes(rawData)
	buffer := util.NewInputBuffer(wire)
	nrs, err := RRsetFromWire(buffer)
	Assert(t, err == nil, "err should be nil")
	matchRRset(t, nrs, rs)
	render := NewMsgRender()
	nrs.Rend(render)
	WireMatch(t, wire, render.Data())
}

func matchRRset(t *testing.T, nrs *RRset, rs *RRset) {
	Assert(t, nrs.Name.Equals(rs.Name), fmt.Sprintf("%s != %s", nrs.Name.String(false), rs.Name.String(false)))
	Equal(t, nrs.Type, rs.Type)
	Equal(t, nrs.Class, rs.Class)
	Equal(t, len(nrs.Rdatas), len(rs.Rdatas))
	for i := 0; i < len(rs.Rdatas); i++ {
		Equal(t, nrs.Rdatas[i].String(), rs.Rdatas[i].String())
	}
}

func TestRRsetFromToWire(t *testing.T) {
	n, _ := NameFromString("test.example.com.")
	ra, _ := AFromString("192.0.2.1")
	matchRRsetRaw(t, "0474657374076578616d706c6503636f6d000001000100000e100004c0000201", &RRset{
		Name:   n,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{ra},
	})
}

func TestRRsetRoateRdata(t *testing.T) {
	ra1, _ := AFromString("1.1.1.1")
	ra2, _ := AFromString("2.2.2.2")
	ra3, _ := AFromString("3.3.3.3")
	n, _ := NameFromString("test.example.com.")
	rrset := &RRset{
		Name:   n,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{ra1},
	}
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra1.String())

	rrset.AddRdata(ra2)
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra2.String())
	Equal(t, rrset.Rdatas[1].String(), ra1.String())
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra1.String())
	Equal(t, rrset.Rdatas[1].String(), ra2.String())

	rrset.AddRdata(ra3)
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra3.String())
	Equal(t, rrset.Rdatas[1].String(), ra1.String())
	Equal(t, rrset.Rdatas[2].String(), ra2.String())
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra2.String())
	Equal(t, rrset.Rdatas[1].String(), ra3.String())
	Equal(t, rrset.Rdatas[2].String(), ra1.String())
	rrset.RotateRdata()
	Equal(t, rrset.Rdatas[0].String(), ra1.String())
	Equal(t, rrset.Rdatas[1].String(), ra2.String())
	Equal(t, rrset.Rdatas[2].String(), ra3.String())

	rrset.RemoveRdata(ra1)
	Equal(t, rrset.RRCount(), 2)
	Equal(t, rrset.Rdatas[0].String(), ra2.String())
	Equal(t, rrset.Rdatas[1].String(), ra3.String())
}

func TestRRsetSortRdata(t *testing.T) {
	ra1, _ := AFromString("1.1.1.1")
	ra2, _ := AFromString("2.2.2.2")
	ra3, _ := AFromString("3.3.3.3")
	n, _ := NameFromString("test.example.com.")
	rrset := &RRset{
		Name:   n,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{ra3, ra2, ra1},
	}

	rrset.SortRdata()
	Equal(t, rrset.Rdatas[0].String(), ra1.String())
	Equal(t, rrset.Rdatas[1].String(), ra2.String())
	Equal(t, rrset.Rdatas[2].String(), ra3.String())
}

func TestRRsetEquals(t *testing.T) {
	ra1, _ := NSFromString("a.com.")
	ra2, _ := NSFromString("b.com.")
	ra3, _ := NSFromString("c.com.")
	n, _ := NameFromString("test.example.com.")
	rrset1 := &RRset{
		Name:   n,
		Type:   RR_NS,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{ra3, ra2, ra1},
	}

	ra4, _ := NSFromString("C.com.")
	rrset2 := &RRset{
		Name:   n,
		Type:   RR_NS,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{ra2, ra4, ra1},
	}
	Assert(t, rrset1.Equals(rrset2), "rrset1 should equl rrset2")
	Assert(t, util.StringSliceCompare([]string{rrset1.String()}, []string{rrset2.String()}, false) != 0, "rrset1 has different rdata order with rrset2")
}

func TestRRsetFromString(t *testing.T) {
	lines := []string{
		"example.com.                      1 IN SOA  ns1.example.com. hostmaster.example.com. 2002022401 10800 15 604800 10800",
		"example.com.                      2 IN NS   ns1.example.com.",
		"example.com.                      4 IN MX   10 mail.example.com.",
		"fred.example.com.                 6 IN A    192.168.0.4",
		"ftp.example.com.                  7 IN CNAME    www.example.com.",
		"1.1.0.10.in-addr.arpa. 100 IN PTR a.com.",
		"naptr.example.com.                8 IN NAPTR 101 10 \"u\" \"sip+E2U\" \"!^.*$!sip:userA@mytest.cn!\" .",
	}

	soaRdata := &SOA{
		MName:   NameFromStringUnsafe("ns1.example.com."),
		RName:   NameFromStringUnsafe("hostmaster.example.com."),
		Serial:  2002022401,
		Refresh: 10800,
		Retry:   15,
		Expire:  604800,
		Minimum: 10800,
	}
	soa := &RRset{Name: NameFromStringUnsafe("example.com."),
		Type:   RR_SOA,
		Class:  CLASS_IN,
		Ttl:    RRTTL(1),
		Rdatas: []Rdata{soaRdata},
	}

	ns := &RRset{Name: NameFromStringUnsafe("example.com."),
		Type:   RR_NS,
		Class:  CLASS_IN,
		Ttl:    RRTTL(2),
		Rdatas: []Rdata{&NS{Name: NameFromStringUnsafe("ns1.example.com.")}},
	}

	mx := &RRset{Name: NameFromStringUnsafe("example.com."),
		Type:   RR_MX,
		Class:  CLASS_IN,
		Ttl:    RRTTL(4),
		Rdatas: []Rdata{&MX{Preference: 10, Exchange: NameFromStringUnsafe("mail.example.com.")}},
	}

	a := &RRset{Name: NameFromStringUnsafe("fred.example.com."),
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(6),
		Rdatas: []Rdata{&A{Host: net.ParseIP("192.168.0.4").To4()}}}

	cname := &RRset{Name: NameFromStringUnsafe("ftp.example.com."),
		Type:   RR_CNAME,
		Class:  CLASS_IN,
		Ttl:    RRTTL(7),
		Rdatas: []Rdata{&CName{Name: NameFromStringUnsafe("www.example.com.")}}}

	ptr := &RRset{Name: NameFromStringUnsafe("1.1.0.10.in-addr.arpa."),
		Type:   RR_PTR,
		Class:  CLASS_IN,
		Ttl:    RRTTL(7),
		Rdatas: []Rdata{&PTR{Name: NameFromStringUnsafe("a.com.")}}}

	naptr := &RRset{Name: NameFromStringUnsafe("naptr.example.com."),
		Type:  RR_NAPTR,
		Class: CLASS_IN,
		Ttl:   RRTTL(8),
		Rdatas: []Rdata{&NAPTR{
			Order:       101,
			Preference:  10,
			Flags:       "u",
			Services:    "sip+E2U",
			Regexp:      "!^.*$!sip:userA@mytest.cn!",
			Replacement: NameFromStringUnsafe("."),
		}},
	}

	expectedRRset := []*RRset{
		soa,
		ns,
		mx,
		a,
		cname,
		ptr,
		naptr,
	}

	for i, line := range lines {
		rrset, err := RRsetFromString(line)
		Assert(t, err == nil, "all rrset is valid %v", err)
		Assert(t, rrset.Equals(expectedRRset[i]), "want [%s] but get [%s]", expectedRRset[i].String(), rrset.String())
	}

	rrsigStr := ".           86400   IN  RRSIG   SOA 8 0 86400 20170522050000 20170509040000 14796 . AwEAAaHIwpx3w4VHKi6i1LHnTaWeHCL154Jug0Rtc9ji5qwPXpBo6A5sRv7cSsPQKPIwxLpyCrbJ4mr2L0EPOdvP6z6YfljK2ZmTbogU9aSU2fiq/4wjxbdkLyoDVgtO+JsxNN4bjr4WcWhsmk1Hg93FV9ZpkWb0Tbad8DFqNDzr//kZ"
	_, err := RRsetFromString(rrsigStr)
	Assert(t, err == nil, "rrsig is valid %v", err)
}
