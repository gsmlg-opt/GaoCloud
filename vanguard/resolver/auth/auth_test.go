package auth

import (
	//	"fmt"
	"testing"

	"cement/domaintree"
	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/httpcmd"
	"vanguard/logger"
	"vanguard/resolver/auth/zone"
	view "vanguard/viewselector"
)

func setupTestZone() *AuthDataSource {
	logger.UseDefaultLogger("error")
	view.InitViews(view.DefaultView)

	conf := &config.VanguardConf{
		Auth: []config.AuthZoneInView{
			config.AuthZoneInView{
				View: "default",
				Zones: []config.AuthZoneConf{
					config.AuthZoneConf{
						Name: "example.com.",
						File: "testdata/example.com",
					},
				},
			},
		},
	}
	return NewAuth(conf)
}

func TestAuthCmd(t *testing.T) {
	auth := setupTestZone()
	originStr := "example.com"
	origin, _ := g53.NameFromString(originStr)
	zoneData, ok := auth.GetZone("default", origin)
	ut.Assert(t, ok != domaintree.NotFound, "should found in mem tree")
	ut.Assert(t, zoneData != nil, "should has value")

	findResult := zoneData.Find(g53.NameFromStringUnsafe(originStr), g53.RR_SOA, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(1))
	findResult = zoneData.Find(g53.NameFromStringUnsafe("joe.tt"), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRNXDomain)

	rrs1 := AuthRRs{&AuthRR{"default", originStr, "aa.example.com", "3600", "A", "1.2.3.4"}}
	rrs2 := AuthRRs{&AuthRR{"default", originStr, "bb.example.com", "3600", "A", "2.2.2.2"}}
	err := auth.addAuthRrs(rrs1)
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	err = auth.addAuthRrs(rrs2)
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	findResult = zoneData.Find(g53.NameFromStringUnsafe("aa.example.com"), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].String(), "1.2.3.4")
	findResult = zoneData.Find(g53.NameFromStringUnsafe("bb.example.com"), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].String(), "2.2.2.2")
	findResult = zoneData.Find(g53.NameFromStringUnsafe(originStr), g53.RR_SOA, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(3))

	newRrs := AuthRRs{&AuthRR{"default", originStr, "aa.example.com", "3600", "A", "6.6.6.6"}}
	err = auth.updateAuthRrs(rrs1, newRrs)
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	findResult = zoneData.Find(g53.NameFromStringUnsafe("aa.example.com"), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].String(), "6.6.6.6")
	findResult = zoneData.Find(g53.NameFromStringUnsafe(originStr), g53.RR_SOA, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(5))

	err = auth.deleteAuthRrs(rrs2)
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	findResult = zoneData.Find(g53.NameFromStringUnsafe(originStr), g53.RR_SOA, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(6))
	findResult = zoneData.Find(g53.NameFromStringUnsafe("bb.example.com"), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRNXDomain)

	rrs3 := AuthRRs{&AuthRR{"default", originStr, "bb.example.com", "3600", "txt", "\"txt test\""}}
	err = auth.addAuthRrs(rrs3)
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	findResult = zoneData.Find(g53.NameFromStringUnsafe(originStr), g53.RR_SOA, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)
	ut.Equal(t, findResult.RRset.Rdatas[0].(*g53.SOA).Serial, uint32(7))
	findResult = zoneData.Find(g53.NameFromStringUnsafe("bb.example.com."), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRNXRRset)

	err = auth.updateAuthZone("default", originStr, []string{"a1"})
	ut.Equal(t, err, (*httpcmd.Error)(nil))

	err = auth.deleteAuthZone("default", originStr)
	ut.Equal(t, err, (*httpcmd.Error)(nil))

	zoneData, ok = auth.GetZone("default", origin)
	ut.Assert(t, ok == domaintree.NotFound, "should not found")
	ut.Assert(t, zoneData == nil, "should no value")
}

func TestAuthInvalidCmd(t *testing.T) {
	auth := setupTestZone()
	cname := &AuthRR{
		View:  view.DefaultView,
		Zone:  "example.com.",
		Name:  "conflict.example.com.",
		Ttl:   "1000",
		Type:  "cname",
		Rdata: "a.cn.",
	}

	a := &AuthRR{
		View:  view.DefaultView,
		Zone:  "example.com.",
		Name:  "conflict.example.com.",
		Ttl:   "1000",
		Type:  "a",
		Rdata: "2.2.2.2",
	}

	_, err := auth.HandleCmd(&AddAuthRrs{
		Rrs: []*AuthRR{
			cname,
			a,
		},
	})
	ut.Assert(t, err != nil, "add a and cname with same name will conflict but get nothing")

	_, err = auth.HandleCmd(&AddAuthRrs{
		Rrs: []*AuthRR{
			a,
			cname,
		},
	})
	ut.Assert(t, err != nil, "add a and cname with same name will conflict but get nothing")

	originStr := "example.com"
	origin, _ := g53.NameFromString(originStr)
	zoneData, _ := auth.GetZone("default", origin)
	findResult := zoneData.Find(g53.NameFromStringUnsafe("conflict.example.com."), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRNXDomain)

	_, err = auth.HandleCmd(&AddAuthRrs{
		Rrs: []*AuthRR{
			cname,
		},
	})
	ut.Assert(t, err == nil, "only add cname should ok")
	findResult = zoneData.Find(g53.NameFromStringUnsafe("conflict.example.com."), g53.RR_CNAME, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)

	_, err = auth.HandleCmd(&AddAuthRrs{
		Rrs: []*AuthRR{
			a,
		},
	})
	ut.Assert(t, err != nil, "add a and cname with same name will conflict but get nothing")

	ns := &AuthRR{
		View:  view.DefaultView,
		Zone:  "example.com.",
		Name:  "example.com.",
		Ttl:   "86400",
		Type:  "ns",
		Rdata: "a.iana-servers.net.",
	}

	old_a := &AuthRR{
		View:  view.DefaultView,
		Zone:  "example.com.",
		Name:  "a.example.com.",
		Ttl:   "3600",
		Type:  "a",
		Rdata: "1.1.1.1",
	}
	_, err = auth.HandleCmd(&DeleteAuthRrs{
		Rrs: []*AuthRR{
			ns,
			old_a,
		},
	})
	ut.Assert(t, err != nil, "delete zone last ns is forbidden")
	findResult = zoneData.Find(g53.NameFromStringUnsafe(old_a.Name), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRSuccess)

	_, err = auth.HandleCmd(&DeleteAuthRrs{
		Rrs: []*AuthRR{
			old_a,
		},
	})
	ut.Assert(t, err == nil, "delete should succeed")
	findResult = zoneData.Find(g53.NameFromStringUnsafe(old_a.Name), g53.RR_A, zone.DefaultFind).GetResult()
	ut.Equal(t, findResult.Type, zone.FRNXDomain)
}
