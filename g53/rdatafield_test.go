package g53

import (
	"g53/util"
	"testing"
)

func TestIPRdataFromAndToWire(t *testing.T) {
	v4_wire, _ := util.HexStrToBytes("c0000201")
	buffer := util.NewInputBuffer(v4_wire)

	rd, l, err := fieldFromWire(RDF_C_IPV4, buffer, 4)
	if err != nil {
		t.Fatalf("from wire failed with %v", err)
	}

	if fieldToString(RDF_D_IP, rd) != "192.0.2.1" {
		t.Errorf("v4 to string failed expect 192.0.2.1 but %v", fieldToString(RDF_D_IP, rd))
	}

	if l != 0 {
		t.Errorf("left len should be 0 but get %v", l)
	}

	out := util.NewOutputBuffer(16)
	fieldToWire(RDF_C_IPV4, rd, out)
	WireMatch(t, v4_wire, out.Data())

	v6_wire, _ := util.HexStrToBytes("20010db8000000000000000000001234")
	buffer = util.NewInputBuffer(v6_wire)

	rd, l, err = fieldFromWire(RDF_C_IPV6, buffer, 16)
	if err != nil {
		t.Fatalf("from wire failed with %v", err)
	}

	if fieldToString(RDF_D_IP, rd) != "2001:db8::1234" {
		t.Errorf("v6 to string failed expect 2001:db8::1234 but %v", fieldToString(RDF_D_IP, rd))
	}

	if l != 0 {
		t.Errorf("left len should be 0 but get %v", l)
	}

	out.Clear()
	fieldToWire(RDF_C_IPV6, rd, out)
	WireMatch(t, v6_wire, out.Data())
}

func TestIntTypeFromToWire(t *testing.T) {
	strs := []string{"77ce5bb9", "00000e10", "0000012c", "0036ee80", "000004b0"}
	results := []string{"2010012601", "3600", "300", "3600000", "1200"}

	for i, str := range strs {
		wire, _ := util.HexStrToBytes(str)
		buffer := util.NewInputBuffer(wire)

		rd, l, err := fieldFromWire(RDF_C_UINT32, buffer, 4)
		if err != nil {
			t.Fatalf("from wire failed with %v", err)
		}

		if fieldToString(RDF_D_INT, rd) != results[i] {
			t.Errorf("v4 to string failed expect %v but %v", results[i], fieldToString(RDF_D_INT, rd))
		}

		if l != 0 {
			t.Errorf("left len should be 0 but get %v", l)
		}

		out := util.NewOutputBuffer(16)
		fieldToWire(RDF_C_UINT32, rd, out)
		WireMatch(t, out.Data(), wire)
	}
}
