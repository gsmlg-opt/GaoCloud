package resolver

import (
	"errors"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/chain"
)

var (
	ErrCNameChainIsTooLong   = errors.New("cname chain is too long")
	ErrCNameCircleIsDetected = errors.New("found cname circle")
)

type CNameHandler struct {
	chain.DefaultResolver
	resolver           chain.Resolver
	checkCnameIndirect bool
}

func NewCNameHandler(resolver chain.Resolver, conf *config.VanguardConf) *CNameHandler {
	return &CNameHandler{
		resolver:           resolver,
		checkCnameIndirect: conf.Resolver.CheckCnameIndirect,
	}
}

func (h *CNameHandler) ReloadConfig(conf *config.VanguardConf) {
	h.checkCnameIndirect = conf.Resolver.CheckCnameIndirect
	chain.ReconfigChain(h.resolver, conf)
}

func (h *CNameHandler) Resolve(client *core.Client) {
	h.resolver.Resolve(client)
	if nextName, redirect := h.needRedirect(client.Response); redirect {
		if h.checkCnameIndirect {
			client.Response.Sections[g53.AnswerSection] = client.Response.Sections[g53.AnswerSection][0:1]
		}
		ctx := newCNameContext(client)
		ctx.addRedirect(nextName)
		h.followCNameChain(ctx)
		ctx.assembleFinalResponse()
	}
}

func (h *CNameHandler) needRedirect(msg *g53.Message) (*g53.Name, bool) {
	if msg == nil {
		return nil, false
	}

	if msg.Header.ANCount == 0 {
		return nil, false
	}

	answers := msg.Sections[g53.AnswerSection]
	index := 0
	if h.checkCnameIndirect == false {
		index = len(answers) - 1
	}
	if rrset := answers[index]; rrset.Type == g53.RR_CNAME {
		return rrset.Rdatas[0].(*g53.CName).Name, true
	} else {
		return nil, false
	}
}

func (h *CNameHandler) followCNameChain(ctx *Context) {
	if err := ctx.makeRedirectQuery(); err != nil {
		return
	}

	h.resolver.Resolve(ctx.client)

	response := ctx.client.Response
	if nextName, redirect := h.needRedirect(response); redirect {
		if h.checkCnameIndirect {
			response.Sections[g53.AnswerSection] = response.Sections[g53.AnswerSection][0:1]
		}
		ctx.mergeResponse(response)
		if err := ctx.addRedirect(nextName); err == nil {
			h.followCNameChain(ctx)
		}
	} else {
		ctx.mergeResponse(response)
	}
}

const maxRedirectCount = 16

type Context struct {
	originalQuestion *g53.Question
	redirectCount    int
	isDone           bool
	response         *g53.Message
	client           *core.Client
	namechain        []*g53.Name
}

func newCNameContext(client *core.Client) *Context {
	return &Context{
		originalQuestion: client.Request.Question,
		client:           client,
		response:         client.Response,
		namechain:        []*g53.Name{client.Request.Question.Name},
	}
}

func (ctx *Context) makeRedirectQuery() error {
	ctx.redirectCount += 1
	if ctx.redirectCount > maxRedirectCount {
		return ErrCNameChainIsTooLong
	}
	ctx.client.Response = nil
	logger.GetLogger().Debug("cname redirect for query %s is detected and query the alias", ctx.client.Request.Question.String())
	ctx.client.Request.Question = &g53.Question{ctx.namechain[len(ctx.namechain)-1],
		ctx.originalQuestion.Type,
		ctx.originalQuestion.Class}
	return nil
}

func (ctx *Context) mergeResponse(response *g53.Message) {
	if response == nil {
		logger.GetLogger().Error("nothing is returned for redirect query %s", ctx.client.Request.Question.String())
		return
	}

	for _, answer := range response.Sections[g53.AnswerSection] {
		ctx.response.AddRRset(g53.AnswerSection, answer)
	}

	ctx.response.Header.Rcode = response.Header.Rcode
}

func (ctx *Context) assembleFinalResponse() {
	request := ctx.client.Request
	request.Question = ctx.originalQuestion
	ctx.response.Header.Id = request.Header.Id
	ctx.response.Question = ctx.originalQuestion
	ctx.client.Response = ctx.response
}

func (ctx *Context) addRedirect(nextName *g53.Name) error {
	for _, name := range ctx.namechain {
		if name.Equals(nextName) {
			logger.GetLogger().Error("cname circle is detected for name %s", nextName.String(false))
			return ErrCNameCircleIsDetected
		}
	}
	ctx.namechain = append(ctx.namechain, nextName)
	return nil
}
