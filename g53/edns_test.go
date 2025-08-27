package g53

import (
	//	"fmt"
	"g53/util"
	"testing"
)

func matchEdns(t *testing.T, rawData string, expectEdns EDNS) {
	edns_wire, _ := util.HexStrToBytes(rawData)
	buf := util.NewInputBuffer(edns_wire)
	edns, err := EdnsFromWire(buf)
	Assert(t, err == nil, "wire data is valid")
	Assert(t, edns.String() == expectEdns.String(), "edns should match")

	render := NewMsgRender()
	edns.Rend(render)
	WireMatch(t, render.Data(), edns_wire)
}

func TestEdnsFromToWire(t *testing.T) {
	matchEdns(t, "0000291000000000000000", EDNS{
		Version:       0,
		extendedRcode: 0,
		UdpSize:       4096,
		DnssecAware:   false,
	})

	matchEdns(t, "0000291000000080000000", EDNS{
		Version:       0,
		extendedRcode: 0,
		UdpSize:       4096,
		DnssecAware:   true,
	})

	matchEdns(t, "0000291000010080000000", EDNS{
		Version:       0,
		extendedRcode: 1,
		UdpSize:       4096,
		DnssecAware:   true,
	})

	matchEdns(t, "00002901ff000080000000", EDNS{
		Version:       0,
		extendedRcode: 0,
		UdpSize:       511,
		DnssecAware:   true,
	})
}
