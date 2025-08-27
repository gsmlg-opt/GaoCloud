package g53

import (
	"bytes"
	"testing"

	"g53/util"
)

func TestTsig(t *testing.T) {
	bindBuf := []byte{186, 134, 40, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 97, 4, 116, 101, 115, 116, 0, 0, 6, 0, 1, 2, 103, 103, 192, 12, 0, 1, 0, 1, 0, 0, 14, 16, 0, 4, 1, 1, 1, 7, 192, 14, 0, 250, 0, 255, 0, 0, 0, 0, 0, 58, 8, 104, 109, 97, 99, 45, 109, 100, 53, 7, 115, 105, 103, 45, 97, 108, 103, 3, 114, 101, 103, 3, 105, 110, 116, 0, 0, 0, 89, 71, 159, 60, 1, 44, 0, 16, 179, 15, 124, 89, 45, 67, 115, 11, 208, 10, 100, 75, 194, 99, 65, 185, 186, 134, 0, 0, 0, 0}
	msg := MakeUpdate(NameFromStringUnsafe("a.test."))
	msg.Header.Id = 47750

	name := NameFromStringUnsafe("gg.a.test.")
	rdata1, _ := AFromString("1.1.1.7")
	rrset := &RRset{
		Name:   name,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{rdata1},
	}

	msg.UpdateAddRRset(rrset)

	tsig, err := NewTSIG("test.", "z08GzEnlCDGy/W3Zw/2NHg==", "hmac-md5")
	Assert(t, err == nil, "create new tsig")
	msg.SetTSIG(tsig)
	msg.Tsig.TimeSigned = uint64(1497866044)

	msg.RecalculateSectionRRCount()
	render := NewMsgRender()
	msg.Rend(render)
	Assert(t, bytes.Equal(render.Data(), bindBuf), "msg with tsig format error")
}

func TestVerify(t *testing.T) {
	msg := MakeUpdate(NameFromStringUnsafe("a.test."))
	msg.Header.Id = 18425

	name := NameFromStringUnsafe("gg.a.test.")
	rdata1, _ := AFromString("1.1.1.7")
	rrset := &RRset{
		Name:   name,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{rdata1},
	}
	msg.UpdateAddRRset(rrset)

	tsig, err := NewTSIG("key_test.", "z08GzEnlCDGy/W3Zw/2NHg==", "hmac-md5")
	Assert(t, err == nil, "msg set tsig failed")
	msg.SetTSIG(tsig)
	render := NewMsgRender()
	msg.RecalculateSectionRRCount()
	msg.Rend(render)
	err = msg.Tsig.VerifyTsig(msg, "z08GzEnlCDGy/W3Zw/2NHg==", nil)
	Assert(t, err == nil, "tsig verify failed")
}

func TestTSIGFromRRset(t *testing.T) {
	msg := MakeUpdate(NameFromStringUnsafe("a.test."))
	msg.Header.Id = 18425

	name := NameFromStringUnsafe("gg.a.test.")
	rdata1, _ := AFromString("1.1.1.7")
	rrset := &RRset{
		Name:   name,
		Type:   RR_A,
		Class:  CLASS_IN,
		Ttl:    RRTTL(3600),
		Rdatas: []Rdata{rdata1},
	}

	msg.UpdateAddRRset(rrset)

	tsig, err := NewTSIG("key_test.", "z08GzEnlCDGy/W3Zw/2NHg==", "hmac-md5")
	Assert(t, err == nil, "tsig create failed")
	msg.SetTSIG(tsig)

	msg.RecalculateSectionRRCount()
	render := NewMsgRender()
	msg.Rend(render)
	msgFromBuf, err := MessageFromWire(util.NewInputBuffer(render.Data()))
	Assert(t, err == nil, "message from wire failed")
	Assert(t, msgFromBuf.Tsig.String() == tsig.String(), "tsig from rrset failed")
}
