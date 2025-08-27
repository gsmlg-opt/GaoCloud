package g53

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"g53/util"
)

type HINFO struct {
	CPU string
	OS  string
}

func (h *HINFO) Rend(r *MsgRender) {
	rendField(RDF_C_BYTE_BINARY, []byte(h.CPU), r)
	rendField(RDF_C_BYTE_BINARY, []byte(h.OS), r)
}

func (h *HINFO) ToWire(buf *util.OutputBuffer) {
	fieldToWire(RDF_C_BYTE_BINARY, []byte(h.CPU), buf)
	fieldToWire(RDF_C_BYTE_BINARY, []byte(h.OS), buf)
}

func (h *HINFO) Compare(other Rdata) int {
	otherHINFO := other.(*HINFO)
	order := fieldCompare(RDF_C_BYTE_BINARY, []byte(h.CPU), []byte(otherHINFO.CPU))
	if order != 0 {
		return order
	}

	return fieldCompare(RDF_C_BYTE_BINARY, []byte(h.OS), []byte(otherHINFO.OS))
}

func (h *HINFO) String() string {
	var buf bytes.Buffer
	buf.WriteString(strings.Join([]string{"\"", fieldToString(RDF_D_STR, h.CPU), "\""}, ""))
	buf.WriteString(" ")
	buf.WriteString(strings.Join([]string{"\"", fieldToString(RDF_D_STR, h.OS), "\""}, ""))
	return buf.String()
}

func HINFOFromWire(buf *util.InputBuffer, ll uint16) (*HINFO, error) {
	c, ll, err := fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}
	c_, _ := c.([]uint8)
	cpu := string(c_)

	o, ll, err := fieldFromWire(RDF_C_BYTE_BINARY, buf, ll)
	if err != nil {
		return nil, err
	}
	o_, _ := o.([]uint8)
	os := string(o_)

	if ll != 0 {
		return nil, errors.New("extra data in rdata part")
	}

	return &HINFO{cpu, os}, nil
}

var hinfoRdataTemplate = regexp.MustCompile(`^\s*(\".*\")\s+(\".*\")\s*$`)

func HINFOFromString(s string) (*HINFO, error) {
	fields := hinfoRdataTemplate.FindStringSubmatch(s)
	if len(fields) != 3 {
		return nil, errors.New("fields count for hinfo isn't 2")
	}

	fields = fields[1:]
	c, err := fieldFromString(RDF_D_STR, fields[0])
	if err != nil {
		return nil, err
	}
	cpu, _ := c.(string)

	o, err := fieldFromString(RDF_D_STR, fields[1])
	if err != nil {
		return nil, err
	}
	os, _ := o.(string)

	return &HINFO{cpu, os}, nil
}
