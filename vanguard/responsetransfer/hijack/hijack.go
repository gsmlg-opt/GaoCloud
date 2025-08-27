package hijack

import (
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	ld "vanguard/localdata"
	"vanguard/logger"
)

type Hijack struct {
	core.DefaultHandler
	rrsets *ld.LocalData
}

func NewHijack() *Hijack {
	h := &Hijack{}
	httpcmd.RegisterHandler(h, []httpcmd.Command{&AddRedirectRR{}, &DeleteRedirectRR{}, &UpdateRedirectRR{}})
	return h
}

func (h *Hijack) ReloadConfig(conf *config.VanguardConf) {
	h.rrsets = ld.NewLocalData()
	for _, c := range conf.Hijack {
		if err := h.rrsets.AddPolicies(c.View, ld.LPLocalRRset, c.Redirect); err != nil {
			panic("invalid redirect:" + err.Error())
		}
	}
}

func (h *Hijack) TransferResponse(client *core.Client) {
	if client.Response != nil && client.Response.Header.Rcode != g53.R_NXDOMAIN {
		return
	}

	if h.rrsets.ResponseWithLocalData(client) {
		logger.GetLogger().Debug("hijack name %s with view %s",
			client.Request.Question.Name.String(false), client.View)
	}
}
