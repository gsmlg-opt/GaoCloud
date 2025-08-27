package dns64

import (
	"sync"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
)

type DNS64 struct {
	core.DefaultHandler
	converters map[string][]*Dns64Converter
	lock       sync.RWMutex
}

func NewDNS64(conf *config.VanguardConf) core.DNSQueryHandler {
	h := &DNS64{}
	h.ReloadConfig(conf)
	httpcmd.RegisterHandler(h, []httpcmd.Command{&PutDns64{}})
	return h
}

func (h *DNS64) ReloadConfig(conf *config.VanguardConf) {
	converters := make(map[string][]*Dns64Converter)
	for _, dns64conf := range conf.DNS64 {
		var viewConverters []*Dns64Converter
		for _, preAndPostfix := range dns64conf.PreAndPostfixes {
			converter, _ := converterFromString(preAndPostfix)
			viewConverters = append(viewConverters, converter)
		}
		if len(viewConverters) > 0 {
			converters[dns64conf.View] = viewConverters
		}
	}
	h.converters = converters
}

func (h *DNS64) HandleQuery(ctx *core.Context) {
	core.PassToNext(h, ctx)
	if h.needDNS64Synthesis(&ctx.Client) {
		h.handleQuery(ctx)
	}
}

func (h *DNS64) needDNS64Synthesis(client *core.Client) bool {
	if h.converters[client.View] == nil {
		return false
	}

	if client.Request.Question.Type != g53.RR_AAAA {
		return false
	}

	if client.Response == nil {
		return false
	}

	if client.Response.Header.Rcode != g53.R_NOERROR {
		return false
	}

	for _, answer := range client.Response.Sections[g53.AnswerSection] {
		if answer.Type == g53.RR_AAAA {
			return false
		}
	}

	return true
}

func (h *DNS64) handleQuery(ctx *core.Context) {
	client := &ctx.Client
	originalResponse := client.Response
	originalQuestion := client.Request.Question
	h.queryARecord(ctx)
	h.synthesizeDNS64Response(client, originalQuestion, originalResponse)
}

func (h *DNS64) queryARecord(ctx *core.Context) {
	client := &ctx.Client
	answers := client.Response.Sections[g53.AnswerSection]
	nextName := client.Request.Question.Name
	if len(answers) > 0 {
		lastRRset := answers[len(answers)-1]
		if lastRRset.Type == g53.RR_CNAME {
			nextName = lastRRset.Rdatas[0].(*g53.CName).Name
		}
	}

	client.Response = nil
	client.Request.Question = &g53.Question{
		nextName,
		g53.RR_A,
		client.Request.Question.Class,
	}
	core.PassToNext(h, ctx)
}

func (h *DNS64) synthesizeDNS64Response(client *core.Client, originalQuestion *g53.Question, originalResponse *g53.Message) {
	client.Request.Question = originalQuestion
	if client.Response == nil {
		client.Response = originalResponse
		return
	}

	finalResponse := originalResponse
	finalAnswers := finalResponse.Sections[g53.AnswerSection]
	answers := client.Response.Sections[g53.AnswerSection]
	h.lock.RLock()
	converters := h.converters[client.View]
	h.lock.RUnlock()
	for _, answer := range answers {
		if answer.Type != g53.RR_A {
			finalAnswers = append(finalAnswers, answer)
		} else {
			finalAnswers = append(finalAnswers, h.synthesisAAAAFromA(converters, answer))
			break
		}
	}
	client.Response.Question = originalQuestion
	client.Response.Sections[g53.AnswerSection] = finalAnswers
}

func (h *DNS64) synthesisAAAAFromA(converters []*Dns64Converter, a *g53.RRset) *g53.RRset {
	var rdatas []g53.Rdata
	for _, convert := range converters {
		for _, rdata := range a.Rdatas {
			rdatas = append(rdatas, &g53.AAAA{convert.synthesisAddr(rdata.(*g53.A).Host)})
		}
	}
	return &g53.RRset{Name: a.Name,
		Type:   g53.RR_AAAA,
		Class:  a.Class,
		Ttl:    a.Ttl,
		Rdatas: rdatas,
	}
}
