package aaaafilter

import (
	"testing"

	ut "cement/unittest"
	"g53"
)

func TestFilterAAAA(t *testing.T) {
	qname := "www.baidu.com"
	request := g53.MakeQuery(g53.NameFromStringUnsafe(qname), g53.RR_A, 512, false)
	response := request.MakeResponse()

	aaaaRdata, _ := g53.AAAAFromString("::1")
	rrset := &g53.RRset{
		Name:   g53.NameFromStringUnsafe(qname),
		Type:   g53.RR_AAAA,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{aaaaRdata},
	}
	response.AddRRset(g53.AnswerSection, rrset)

	aRdata, _ := g53.AFromString("127.0.0.1")
	rrset = &g53.RRset{
		Name:   g53.NameFromStringUnsafe(qname),
		Type:   g53.RR_A,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{aRdata},
	}
	response.AddRRset(g53.AnswerSection, rrset)
	response.Header.ANCount = 2

	aaaaRdata, _ = g53.AAAAFromString("::2")
	rrset = &g53.RRset{
		Name:   g53.NameFromStringUnsafe(qname),
		Type:   g53.RR_AAAA,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{aaaaRdata},
	}
	response.AddRRset(g53.AdditionalSection, rrset)

	filter := &aaaaFilter{}
	filter.removeAaaaRecords(response)

	ut.Equal(t, len(response.Sections[g53.AnswerSection]), 1)
	ut.Equal(t, len(response.Sections[g53.AdditionalSection]), 0)
	ut.Equal(t, response.Sections[g53.AnswerSection][0].Type, g53.RR_A)
}
