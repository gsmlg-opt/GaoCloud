package g53

import (
	"g53/util"
	"testing"
)

func TestWriteName(t *testing.T) {
	raw, _ := util.HexStrToBytes("0161076578616d706c6503636f6d000162c0020161076578616d706c65036f726700")
	render := NewMsgRender()
	aExampleCom, _ := NewName("a.example.com", true)
	bExampleCom, _ := NewName("b.example.com", true)
	aExampleOrg, _ := NewName("a.example.org", true)
	render.WriteName(aExampleCom, true)
	render.WriteName(bExampleCom, true)
	render.WriteName(aExampleOrg, true)
	WireMatch(t, raw, render.Data())

	raw, _ = util.HexStrToBytes("0161076578616d706c6503636f6d00ffff0162076578616d706c6503636f6d00")
	render.Clear()
	offset := uint(0x3fff)
	render.Skip(offset)
	render.WriteName(aExampleCom, true)
	render.WriteName(aExampleCom, true)
	render.WriteName(bExampleCom, true)
	WireMatch(t, raw, render.Data()[offset:])

	raw, _ = util.HexStrToBytes("0161076578616d706c6503636f6d000162076578616d706c6503636f6d00c00f")
	render.Clear()
	render.WriteName(aExampleCom, true)
	render.WriteName(bExampleCom, false)
	render.WriteName(bExampleCom, true)
	WireMatch(t, raw, render.Data())

	raw, _ = util.HexStrToBytes("0161076578616d706c6503636f6d000162c002c00f")
	render.Clear()
	render.WriteName(aExampleCom, true)
	render.WriteName(bExampleCom, true)
	render.WriteName(bExampleCom, true)
	WireMatch(t, raw, render.Data())

	raw, _ = util.HexStrToBytes("0161076578616d706c6503636f6d000162c0020161076578616d706c65036f726700")
	render.Clear()
	bExampleComCS, _ := NewName("b.exAmple.CoM", false)
	render.WriteName(aExampleCom, true)
	render.WriteName(bExampleComCS, true)
	render.WriteName(aExampleOrg, true)
	WireMatch(t, raw, render.Data())
}

func BenchmarkRenderWriteName(b *testing.B) {
	aExampleCom, _ := NewName("a.example.com", true)
	bExampleCom, _ := NewName("b.Example.com", true)
	render := NewMsgRender()
	for i := 0; i < b.N; i++ {
		render.WriteName(aExampleCom, true)
		render.WriteName(bExampleCom, true)
		render.Clear()
	}
}
