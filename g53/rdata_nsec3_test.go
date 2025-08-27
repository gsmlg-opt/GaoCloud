package g53

import (
	"testing"

	"g53/util"
)

/*
*1 1 0 - CK0Q1GIN43N1ARRC9OSM6QPQR81H5M9A NS SOA RRSIG DNSKEY NSEC3PARAM
 */
func TestNSEC3FromWire(t *testing.T) {
	nsec3_wire, _ := util.HexStrToBytes("00230101000000146501a0c25720ee156f6c4e39636b3ada0312d92a000722000000000290")
	buf := util.NewInputBuffer(nsec3_wire)

	rr, err := RdataFromWire(RR_NSEC3, buf)
	if err != nil {
		t.Fatalf("nsec3 from wire failed with %v", err)
	}

	nsec3, ok := rr.(*NSEC3)
	if ok == false {
		t.Fatalf("parse from wire should be nsec3 but not")
	}

	Assert(t, nsec3.Algorithm == 1, "nsec3 Algorithm should be 1")
	Assert(t, nsec3.Flags == 1, "nsec3 Flags should be 1")
	Assert(t, nsec3.Iterations == 0, "nsec3 Iterations should be 0")
	Assert(t, nsec3.SaltLength == 0, "nsec3 SaltLen should be 0")
	Assert(t, nsec3.Salt == "", "nsec3 Salt should be empty")
	Assert(t, nsec3.NextHash == "CK0Q1GIN43N1ARRC9OSM6QPQR81H5M9A",
		"nsec3 NextHash should be CK0Q1GIN43N1ARRC9OSM6QPQR81H5M9A")
	Assert(t, nsec3.Types[0] == RR_NS, "nsec3 Types[0] should be NS")
	Assert(t, nsec3.Types[1] == RR_SOA, "nsec3 Types[1] should be SOA")
	Assert(t, nsec3.Types[2] == RR_RRSIG, "nsec3 Types[2] should be RRSIG")
	Assert(t, nsec3.Types[3] == RR_DNSKEY, "nsec3 Types[3] should be DNSKEY")
	Assert(t, nsec3.Types[4] == RR_NSEC3PARAM, "nsec3 Types[4] should be NSEC3PARAM")

	render := NewMsgRender()
	render.WriteUint16(35)
	nsec3.Rend(render)
	WireMatch(t, render.Data(), nsec3_wire)
}
