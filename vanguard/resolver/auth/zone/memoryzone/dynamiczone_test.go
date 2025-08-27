package memoryzone

import (
	"fmt"
	"sort"
	"testing"

	ut "cement/unittest"
	"g53"
	"vanguard/logger"
	zn "vanguard/resolver/auth/zone"
)

var dynamicZoneData []string = []string{
	"cn. 300 IN SOA a.dns.cn. root.cnnic.cn. 2023300522 7200 3600 2419200 21600",
	"cn. 300 IN NS ns.cn.",
	"ns.cn. 300 IN A 1.1.1.1",
	"a.cn. 300 IN A 1.1.1.1",
	"b.cn. 300 IN A 1.1.1.1",
	"c.cn. 300 IN A 1.1.1.1",
	"d.cn. 300 IN A 1.1.1.1",
	"f.cn. 300 IN A 2.2.2.2",
	"a.a.a.cn. 300 IN A 1.1.1.1",
	"b.a.a.cn. 300 IN A 1.1.1.1",
}

func createDynamicZone(name string, data []string) *DynamicZone {
	zone := NewDynamicZone(g53.NameFromStringUnsafe(name))
	tx, _ := zone.Begin()
	for _, rr := range data {
		rrset, err := g53.RRsetFromString(rr)
		if err != nil {
			panic(fmt.Sprintf("rr %s isn't valid %vs", rr, err.Error()))
		}
		zone.Add(tx, rrset)
	}
	tx.Commit()
	return zone
}

func zoneHasARRset(t *testing.T, zone *DynamicZone, name string, address []string) {
	ctx := zone.Find(g53.NameFromStringUnsafe(name), g53.RR_A, zn.DefaultFind)
	result := ctx.GetResult()
	if len(address) == 0 {
		ut.Assert(t, result.Type != zn.FRSuccess, "no data should be found")
	} else {
		ut.Equal(t, result.Type, zn.FRSuccess)
		ut.Equal(t, result.RRset.RRCount(), len(address))

		getAddress := make([]string, 0, len(address))
		for _, rdata := range result.RRset.Rdatas {
			getAddress = append(getAddress, rdata.String())
		}
		sort.Strings(address)
		sort.Strings(getAddress)
		ut.Equal(t, getAddress, address)
	}
}

func TestDynamicClean(t *testing.T) {
	logger.UseDefaultLogger("error")
	dzone := createDynamicZone("cn", dynamicZoneData)

	result := dzone.Find(g53.NameFromStringUnsafe("a.a.cn."), g53.RR_A, zn.DefaultFind).GetResult()
	ut.Equal(t, result.Type, zn.FRNXRRset)

	tx, _ := dzone.Begin()
	for _, name := range []string{"a.a.a.cn", "b.a.a.cn"} {
		dzone.DeleteRRset(tx, &g53.RRset{
			Name: g53.NameFromStringUnsafe(name),
			Type: g53.RR_A,
		})
	}
	tx.Commit()

	ctx := dzone.Find(g53.NameFromStringUnsafe("a.a.cn."), g53.RR_A, zn.DefaultFind)
	result = ctx.GetResult()
	ut.Equal(t, result.Type, zn.FRNXDomain)

	tx, _ = dzone.Begin()
	dzone.DeleteRRset(tx, &g53.RRset{
		Name: g53.NameFromStringUnsafe("d.cn."),
		Type: g53.RR_A,
	})
	tx.Commit()

	ut.Equal(t, dzone.DomainCount(), 6)
	zoneHasARRset(t, dzone, "f.cn.", []string{"2.2.2.2"})
}

func TestTransaction(t *testing.T) {
	logger.UseDefaultLogger("error")
	dzone := createDynamicZone("cn", dynamicZoneData)

	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1"})
	tx, _ := dzone.Begin()
	rrset, _ := g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.Add(tx, rrset)
	tx.RollBack()
	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1"})

	tx, _ = dzone.Begin()
	rrset, _ = g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.Add(tx, rrset)
	tx.Commit()
	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1", "2.2.2.2"})

	tx, _ = dzone.Begin()
	rrset, _ = g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.DeleteRr(tx, rrset)
	tx.RollBack()
	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1", "2.2.2.2"})

	tx, _ = dzone.Begin()
	rrset, _ = g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.DeleteRr(tx, rrset)
	tx.Commit()
	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1"})

	tx, _ = dzone.Begin()
	rrset, _ = g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.DeleteRRset(tx, rrset)
	tx.RollBack()
	zoneHasARRset(t, dzone, "a.cn.", []string{"1.1.1.1"})

	tx, _ = dzone.Begin()
	rrset, _ = g53.RRsetFromString("a.cn. 300 IN A 2.2.2.2")
	dzone.DeleteRRset(tx, rrset)
	tx.Commit()
	zoneHasARRset(t, dzone, "a.cn.", []string{})
}
