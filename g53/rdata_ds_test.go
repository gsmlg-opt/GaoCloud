package g53

import (
	"strings"
	"testing"

	"g53/util"
)

/*
* 30909 8 2 E2D3C916F6DEEAC73294E8268FB5885044A833FC5459588F4A9184CFC41A5766
 */
func TestDSFromWire(t *testing.T) {
	ds_wire, _ := util.HexStrToBytes("002478bd0802e2d3c916f6deeac73294e8268fb5885044a833fc5459588f4a9184cfc41a5766")
	buf := util.NewInputBuffer(ds_wire)

	rr, err := RdataFromWire(RR_DS, buf)
	if err != nil {
		t.Fatalf("ds from wire failed with %v", err)
	}

	ds, ok := rr.(*DS)
	if ok == false {
		t.Fatalf("parse from wire should be ds but not")
	}

	Assert(t, ds.KeyTag == 30909, "ds KeyTag should be 30909")
	Assert(t, ds.Algorithm == 8, "ds Algorithm should be 8")
	Assert(t, ds.DigestType == 2, "ds DigestType should be 2")
	Assert(t, strings.ToUpper(ds.Digest) == "E2D3C916F6DEEAC73294E8268FB5885044A833FC5459588F4A9184CFC41A5766",
		"ds Digest should be E2D3C916F6DEEAC73294E8268FB5885044A833FC5459588F4A9184CFC41A5766")

	render := NewMsgRender()
	render.WriteUint16(36)
	ds.Rend(render)
	WireMatch(t, render.Data(), ds_wire)
}
