package k8s

import (
	"g53"
	"vanguard/config"
	"vanguard/core"
)

type K8SAuth struct {
	core.DefaultHandler

	auth       *Auth
	controller *Controller
}

func NewK8SAuth(cfg *config.VanguardConf) core.DNSQueryHandler {
	auth, err := NewAuth(cfg)
	if err != nil {
		panic("create k8s module failed:" + err.Error())
	}

	controller, err := NewK8sController(auth)
	if err != nil {
		panic("create k8s module failed:" + err.Error())
	}

	return &K8SAuth{
		auth:       auth,
		controller: controller,
	}
}

func (a *K8SAuth) HandleQuery(ctx *core.Context) {
	if a.auth.resolve(&ctx.Client) == false {
		core.PassToNext(a, ctx)
		return
	}

	if isCNameResponse(ctx.Client.Response) {
		a.redirectQuery(ctx)
	}
}

func isCNameResponse(msg *g53.Message) bool {
	if msg.Header.ANCount != 1 {
		return false
	}

	answers := msg.Sections[g53.AnswerSection]
	rrset := answers[0]
	return rrset.Type == g53.RR_CNAME
}

func (a *K8SAuth) redirectQuery(ctx *core.Context) {
	client := &ctx.Client
	originResp := client.Response
	originalName := client.Request.Question.Name

	client.Response = nil
	client.Request.Question.Name = originResp.Sections[g53.AnswerSection][0].Rdatas[0].(*g53.CName).Name
	core.PassToNext(a, ctx)
	if client.Response != nil {
		client.Response = mergeResponse(originResp, client.Response)
		client.Request.Question.Name = originalName
	}
}

func mergeResponse(firstResp *g53.Message, redirectResp *g53.Message) *g53.Message {
	for _, answer := range redirectResp.Sections[g53.AnswerSection] {
		firstResp.AddRRset(g53.AnswerSection, answer)
	}
	firstResp.Header.Rcode = redirectResp.Header.Rcode
	return firstResp
}
