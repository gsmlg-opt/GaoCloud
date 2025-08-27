package dns64

import (
	"net"
	"testing"

	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
)

type dumbHander struct {
	core.DefaultHandler
	index    int
	response []*g53.Message
}

func (dumb *dumbHander) HandleQuery(ctx *core.Context) {
	client := &ctx.Client
	response := *dumb.response[dumb.index]
	client.Response = &response
	dumb.index += 1
}

func buildCNameRRset(source, target string) *g53.RRset {
	cnameRdata, _ := g53.CNameFromString(target)
	return &g53.RRset{
		Name:   g53.NameFromStringUnsafe(source),
		Type:   g53.RR_CNAME,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{cnameRdata},
	}
}

func buildARRset(qname, addr string) *g53.RRset {
	ip, _ := g53.AFromString(addr)
	return &g53.RRset{
		Name:   g53.NameFromStringUnsafe(qname),
		Type:   g53.RR_A,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(3600),
		Rdatas: []g53.Rdata{ip},
	}
}

func buildResponse(qname string, qtype g53.RRType, answers ...*g53.RRset) *g53.Message {
	request := g53.MakeQuery(g53.NameFromStringUnsafe(qname), qtype, 512, false)
	response := request.MakeResponse()
	for _, answer := range answers {
		response.AddRRset(g53.AnswerSection, answer)
		response.Header.ANCount += 1
	}
	return response
}

func TestDns64SyncResponse(t *testing.T) {
	logger.UseDefaultLogger("debug")
	dumb := &dumbHander{}
	dumb.response = []*g53.Message{
		buildResponse("a1.cn.", g53.RR_AAAA),
		buildResponse("a1.cn.", g53.RR_A, buildCNameRRset("a1.cn.", "b1.cn."), buildARRset("b1.cn.", "192.0.2.1")),
	}

	d64 := NewDNS64(&config.VanguardConf{}).(*DNS64)
	d64.SetNext(dumb)
	converter, _ := converterFromString("64:ff9b::/96 ")
	d64.converters["v1"] = []*Dns64Converter{converter}
	ctx := core.NewContext()
	ctx.Client.Request = g53.MakeQuery(g53.NameFromStringUnsafe("a1.cn."), g53.RR_AAAA, 512, false)
	ctx.Client.View = "v1"
	d64.HandleQuery(ctx)

	request := ctx.Client.Request
	response := ctx.Client.Response
	ut.Assert(t, request.Question.Equals(response.Question), "request question should same with response question")
	answers := response.Sections[g53.AnswerSection]
	ut.Equal(t, len(answers), 2)
	ut.Equal(t, answers[0].Type, g53.RR_CNAME)
	ut.Equal(t, answers[1].Type, g53.RR_AAAA)
	v6Expect := net.ParseIP("64:ff9b::192.0.2.1")
	ut.Equal(t, answers[1].Rdatas[0].(*g53.AAAA).Host, v6Expect)
}
