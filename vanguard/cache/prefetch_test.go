package cache

import (
	"testing"
	"time"

	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
)

type dumbResolver struct {
	respIP string
}

func (r *dumbResolver) Next() core.DNSQueryHandler {
	return r
}

func (r *dumbResolver) SetNext(core.DNSQueryHandler) {
}

func (r *dumbResolver) HandleQuery(ctx *core.Context) {
	client := &ctx.Client
	resp := client.Request.MakeResponse()
	rdata, _ := g53.AFromString(r.respIP)
	resp.AddRR(g53.AnswerSection,
		client.Request.Question.Name,
		client.Request.Question.Type,
		client.Request.Question.Class,
		g53.RRTTL(12),
		rdata,
		false)
	client.Response = resp
	client.CacheAnswer = true
}

func TestPrefetch(t *testing.T) {
	logger.UseDefaultLogger("debug")
	conf := &config.CacheConf{
		PositiveTtl:  60,
		NegativeTtl:  60,
		MaxCacheSize: uint(3),
		ShortAnswer:  true,
		Prefetch:     true,
	}

	resolver := &dumbResolver{
		respIP: "2.2.2.2",
	}
	cache := newMessageCache(conf, resolver)
	message := buildMessage("test.example.com.", "1.1.1.1", 12)
	cache.Add(message)

	qname, _ := g53.NameFromString("test.example.com.")
	client := &core.Client{
		Request: g53.MakeQuery(qname, g53.RR_A, 512, false),
	}
	message, found := cache.Get(client)
	ut.Assert(t, found == true, "message shouldn't expired")
	ut.Equal(t, message.Sections[g53.AnswerSection][0].Rdatas[0].String(), "1.1.1.1")

	<-time.After(3 * time.Second)
	message, found = cache.Get(client)
	ut.Assert(t, found == true, "message shouldn't expired")
	ut.Equal(t, message.Sections[g53.AnswerSection][0].Rdatas[0].String(), "1.1.1.1")

	<-time.After(1 * time.Second)
	message, found = cache.Get(client)
	ut.Assert(t, found == true, "message shouldn't expired")
	ut.Equal(t, message.Sections[g53.AnswerSection][0].Rdatas[0].String(), "2.2.2.2")
}
