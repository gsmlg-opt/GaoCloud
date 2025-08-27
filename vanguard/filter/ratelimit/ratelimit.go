package ratelimit

import (
	"vanguard/config"
	"vanguard/core"
)

type RateLimit struct {
	ip_throtter   *IPThrottler
	name_throtter *NameThrottler
}

func NewRateLimit(conf *config.VanguardConf) *RateLimit {
	return &RateLimit{
		ip_throtter:   NewIPThrotter(conf),
		name_throtter: NewNameThrotter(conf),
	}
}

func (limit *RateLimit) ReloadConfig(conf *config.VanguardConf) {
	limit.ip_throtter.ReloadConfig(conf)
	limit.name_throtter.ReloadConfig(conf)
}

func (limit *RateLimit) AllowQuery(ctx *core.Context) bool {
	client := &ctx.Client
	return limit.ip_throtter.IsIPAllowed(client.IP()) &&
		(client.Request.Question != nil &&
			limit.name_throtter.IsNameAllowed(client.View, client.Request.Question.Name))
}
