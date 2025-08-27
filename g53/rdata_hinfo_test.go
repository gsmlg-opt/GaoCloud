package g53

import (
	"testing"

	"g53/util"
)

func TestRRsetFromStringWithHINFO(t *testing.T) {
	hinfostr := "www.baidu.com. 3600 IN HINFO \"Petium II 266\" \"Redhat 7.1\""
	rrset, err := RRsetFromString(hinfostr)
	Assert(t, err == nil, "err should be nil")

	hinfo, err := HINFOFromString("\"Petium II 266\" \"Redhat 7.1\"")
	Assert(t, err == nil, "err should be nil")

	expectRRset := &RRset{
		Name:   NameFromStringUnsafe("www.baidu.com."),
		Type:   RR_HINFO,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{hinfo},
	}
	Assert(t, rrset.Equals(expectRRset), "rrset should be equals expect rrset")
}

//"Intel 126" "CentOS 7.6"
func TestHINFOFromWire(t *testing.T) {
	hinfo_wire, _ := util.HexStrToBytes("001509496e74656c203132360a43656e744f5320372e36")
	buf := util.NewInputBuffer(hinfo_wire)
	hinfo, err := RdataFromWire(RR_HINFO, buf)
	Assert(t, err == nil, "err should be nil")

	Assert(t, hinfo.(*HINFO).CPU == "Intel 126", "HINFO CPU should be \"Intel 126\"")
	Assert(t, hinfo.(*HINFO).OS == "CentOS 7.6", "HINFO OS should be \"CentOS 7.6\"")
	render := NewMsgRender()
	render.WriteUint16(21)
	hinfo.Rend(render)
	WireMatch(t, render.Data(), hinfo_wire)
}
