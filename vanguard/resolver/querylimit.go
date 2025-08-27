package resolver

import (
	"cement/singleflight"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/chain"

	"g53"
)

type QueryLimit struct {
	chain.DefaultResolver
	resolver      chain.Resolver
	outQueryGroup *singleflight.Group
}

func NewQueryLimit(resolver chain.Resolver, conf *config.VanguardConf) *QueryLimit {
	limit := &QueryLimit{resolver: resolver}
	limit.reloadConfig(conf)
	return limit
}

func (limit *QueryLimit) ReloadConfig(conf *config.VanguardConf) {
	limit.reloadConfig(conf)
	chain.ReconfigChain(limit.resolver, conf)
}

func (limit *QueryLimit) reloadConfig(conf *config.VanguardConf) {
	limit.outQueryGroup = singleflight.New(uint32(conf.Server.HandlerCount * 8 / 10))
}

type resolveResponse struct {
	resp        *g53.Message
	cacheAnswer bool
}

func (limit *QueryLimit) Resolve(client *core.Client) {
	r_, err := limit.outQueryGroup.Do(client.QueryKey(), func() (interface{}, error) {
		limit.resolver.Resolve(client)
		return resolveResponse{client.Response, client.CacheAnswer}, nil
	})

	if err != nil {
		logger.GetLogger().Error("out query exceed limit:" + err.Error())
		return
	}

	if client.Response != nil {
		return
	}

	r := r_.(resolveResponse)
	if r.resp == nil {
		return
	}

	if client.Request.Question.Name.Equals(r.resp.Question.Name) {
		respCopy := *r.resp
		respCopy.Header.Id = client.Request.Header.Id
		client.Response = &respCopy
		client.CacheAnswer = r.cacheAnswer
	} else {
		limit.Resolve(client)
	}
}
