package srvfailedprotector

import (
	"sync/atomic"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
)

type SFProtector struct {
	drop int32
}

func NewSFProtector(conf *config.VanguardConf) *SFProtector {
	p := &SFProtector{}
	p.ReloadConfig(conf)
	httpcmd.RegisterHandler(p, []httpcmd.Command{&SrvFailedProtect{}})
	return p
}

func (p *SFProtector) ReloadConfig(conf *config.VanguardConf) {
	if conf.Filter.DropSrvFailed {
		p.drop = 1
	} else {
		p.drop = 0
	}
}

func (p *SFProtector) AllowResponse(ctx *core.Context) bool {
	if atomic.LoadInt32(&p.drop) == 0 {
		return true
	}

	response := ctx.Client.Response
	return response != nil && response.Header.Rcode != g53.R_SERVFAIL
}
