package resolver

import (
	"testing"

	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/chain"
)

type dumbHander struct {
	chain.DefaultResolver
	index    int
	response []*g53.Message
	t        *testing.T
}

func (dumb *dumbHander) Resolve(client *core.Client) {
	response := *dumb.response[dumb.index]
	ut.Assert(dumb.t, client.Request.Question.Name.Equals(response.Question.Name),
		"request question name "+client.Request.Question.Name.String(false)+
			" diff from resp question name "+response.Question.Name.String(false))

	client.Response = &response
	dumb.index += 1
}

func (dumb *dumbHander) ReloadConfig(conf *config.VanguardConf) {
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

func buildResponse(qname string, answer *g53.RRset) *g53.Message {
	request := g53.MakeQuery(g53.NameFromStringUnsafe(qname), answer.Type, 512, false)
	response := request.MakeResponse()
	response.AddRRset(g53.AnswerSection, answer)
	response.Header.ANCount = 1
	return response
}

func TestCNameHandleNonCNameResult(t *testing.T) {
	logger.UseDefaultLogger("error")
	dumb := &dumbHander{t: t}
	conf := &config.VanguardConf{
		Resolver: config.ResolverConf{
			CheckCnameIndirect: false,
		},
	}
	handler := NewCNameHandler(dumb, conf)

	request := g53.MakeQuery(g53.NameFromStringUnsafe("a1.cn."), g53.RR_A, 512, false)
	var client core.Client
	client.Request = request
	client.View = "v1"
	dumb.response = []*g53.Message{buildResponse("a1.cn.", buildARRset("a1.cn.", "1.1.1.1"))}
	handler.Resolve(&client)

	ut.Assert(t, client.Response.Sections[g53.AnswerSection][0].Equals(dumb.response[0].Sections[g53.AnswerSection][0]), "")
}

func TestCNameHandleCNameChain(t *testing.T) {
	logger.UseDefaultLogger("error")
	dumb := &dumbHander{t: t}
	conf := &config.VanguardConf{
		Resolver: config.ResolverConf{
			CheckCnameIndirect: false,
		},
	}
	handler := NewCNameHandler(dumb, conf)

	request := g53.MakeQuery(g53.NameFromStringUnsafe("a1.cn."), g53.RR_A, 512, false)
	var client core.Client
	client.Request = request
	client.View = "v1"

	dumb.response = []*g53.Message{
		buildResponse("a1.cn.", buildCNameRRset("a1.cn.", "a2.cn.")),
		buildResponse("a2.cn.", buildCNameRRset("a2.cn.", "a3.cn.")),
		buildResponse("a3.cn.", buildCNameRRset("a3.cn.", "a4.cn.")),
		buildResponse("a4.cn.", buildCNameRRset("a4.cn.", "a5.cn.")),
		buildResponse("a5.cn.", buildARRset("a5.cn.", "2.2.2.2")),
	}

	handler.Resolve(&client)

	answers := client.Response.Sections[g53.AnswerSection]
	ut.Equal(t, len(answers), 5)
	for i := 0; i < 5; i++ {
		ut.Assert(t, answers[i].Equals(dumb.response[i].Sections[g53.AnswerSection][0]), "")
	}
}

func TestCNameInDirect(t *testing.T) {
	logger.UseDefaultLogger("error")
	dumb := &dumbHander{t: t}
	conf := &config.VanguardConf{
		Resolver: config.ResolverConf{
			CheckCnameIndirect: true,
		},
	}
	handler := NewCNameHandler(dumb, conf)

	request := g53.MakeQuery(g53.NameFromStringUnsafe("a1.cn."), g53.RR_A, 512, false)
	var client core.Client
	client.Request = request
	client.View = "v1"

	resp1 := buildResponse("a1.cn.", buildCNameRRset("a1.cn.", "a2.cn."))
	resp1.AddRRset(g53.AnswerSection, buildCNameRRset("a2.cn.", "a3.cn."))
	resp1.AddRRset(g53.AnswerSection, buildARRset("a3.cn.", "1.1.1.1"))
	resp2 := buildResponse("a2.cn.", buildCNameRRset("a2.cn.", "a4.cn."))
	resp2.AddRRset(g53.AnswerSection, buildARRset("a4.cn.", "2.2.2.2"))
	dumb.response = []*g53.Message{
		resp1,
		resp2,
		buildResponse("a4.cn.", buildARRset("a4.cn.", "3.3.3.3")),
	}

	handler.Resolve(&client)

	answers := client.Response.Sections[g53.AnswerSection]
	ut.Equal(t, len(answers), 3)
	ut.Equal(t, answers[0].Rdatas[0].String(), "a2.cn.")
	ut.Equal(t, answers[1].Rdatas[0].String(), "a4.cn.")
	ut.Equal(t, answers[2].Rdatas[0].String(), "3.3.3.3")
}

func TestCNameReDirect(t *testing.T) {
	logger.UseDefaultLogger("error")
	dumb := &dumbHander{t: t}
	conf := &config.VanguardConf{
		Resolver: config.ResolverConf{
			CheckCnameIndirect: false,
		},
	}
	handler := NewCNameHandler(dumb, conf)

	request := g53.MakeQuery(g53.NameFromStringUnsafe("a1.cn."), g53.RR_A, 512, false)
	var client core.Client
	client.Request = request
	client.View = "v1"

	resp1 := buildResponse("a1.cn.", buildCNameRRset("a1.cn.", "a2.cn."))
	resp1.AddRRset(g53.AnswerSection, buildCNameRRset("a2.cn.", "a3.cn."))
	resp2 := buildResponse("a3.cn.", buildCNameRRset("a3.cn.", "a4.cn."))
	resp2.AddRRset(g53.AnswerSection, buildCNameRRset("a4.cn.", "a5.cn."))
	dumb.response = []*g53.Message{
		resp1,
		resp2,
		buildResponse("a5.cn.", buildARRset("a5.cn.", "5.5.5.5")),
	}

	handler.Resolve(&client)

	answers := client.Response.Sections[g53.AnswerSection]
	ut.Equal(t, len(answers), 5)
	ut.Equal(t, answers[0].Rdatas[0].String(), "a2.cn.")
	ut.Equal(t, answers[1].Rdatas[0].String(), "a3.cn.")
	ut.Equal(t, answers[2].Rdatas[0].String(), "a4.cn.")
	ut.Equal(t, answers[3].Rdatas[0].String(), "a5.cn.")
	ut.Equal(t, answers[4].Rdatas[0].String(), "5.5.5.5")
}
