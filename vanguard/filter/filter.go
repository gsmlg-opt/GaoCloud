package filter

import (
	"vanguard/config"
	"vanguard/core"

	"vanguard/filter/ratelimit"
	"vanguard/filter/srvfailedprotector"
)

type PreFilter interface {
	AllowQuery(*core.Context) bool
}

type PostFilter interface {
	AllowResponse(*core.Context) bool
}

type FilterChain struct {
	core.DefaultHandler
	preFilters  []PreFilter
	postFilters []PostFilter
}

func NewFilterChain(conf *config.VanguardConf) core.DNSQueryHandler {
	c := &FilterChain{}
	c.AddPreFilter(ratelimit.NewRateLimit(conf))
	c.AddPostFilter(srvfailedprotector.NewSFProtector(conf))
	return c
}

func (c *FilterChain) AddPreFilter(f PreFilter) {
	c.preFilters = append(c.preFilters, f)
}

func (c *FilterChain) AddPostFilter(f PostFilter) {
	c.postFilters = append(c.postFilters, f)
}

func (c *FilterChain) HandleQuery(ctx *core.Context) {
	for _, f := range c.preFilters {
		if f.AllowQuery(ctx) == false {
			return
		}
	}

	core.PassToNext(c, ctx)

	for _, f := range c.postFilters {
		if f.AllowResponse(ctx) == false {
			ctx.Client.Response = nil
			return
		}
	}
}

func (c *FilterChain) ReloadConfig(conf *config.VanguardConf) {
	for _, f := range c.preFilters {
		config.ReloadConfig(f, conf)
	}

	for _, f := range c.postFilters {
		config.ReloadConfig(f, conf)
	}
}
