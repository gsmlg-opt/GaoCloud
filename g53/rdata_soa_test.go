package g53

import (
	"g53/util"
	"testing"
)

func TestSOAFromToWire(t *testing.T) {
	soa_wire, _ := util.HexStrToBytes("002b026e73076578616d706c6503636f6d0004726f6f74c00577ce5bb900000e100000012c0036ee80000004b0")
	buf := util.NewInputBuffer(soa_wire)

	rr, err := RdataFromWire(RR_SOA, buf)
	if err != nil {
		t.Fatalf("soa from wire failed with %v", err)
	}

	soa, ok := rr.(*SOA)
	if ok == false {
		t.Fatalf("parse from wire should be soa but not")
	}

	NameEqToStr(t, soa.MName, "ns.example.com.")

	render := NewMsgRender()
	render.WriteUint16(43)
	soa.Rend(render)
	WireMatch(t, render.Data(), soa_wire)
}
