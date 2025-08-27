package forwarder

import (
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/chain"
	"vanguard/resolver/querysource"
	"vanguard/util"
)

type Forwarder struct {
	chain.DefaultResolver
	viewFwder *ViewFwderMgr
}

func NewForwarder(conf *config.VanguardConf) *Forwarder {
	fwd := &Forwarder{
		viewFwder: NewViewFwderMgr(conf),
	}
	fwd.ReloadConfig(conf)
	return fwd
}

func (fwder *Forwarder) ReloadConfig(conf *config.VanguardConf) {
	fwder.viewFwder.ReloadConfig(conf)
}

func (fwder *Forwarder) Resolve(client *core.Client) {
	if client.Response != nil && util.ClassifyResponse(client.Response) == util.REFERRAL {
		return
	}

	fwder.processRequest(client)
	if client.Response != nil {
		client.Response.Header.Id = client.Request.Header.Id
		client.Response.Header.SetFlag(g53.FLAG_AA, false)
		client.CacheAnswer = true
	} else {
		chain.PassToNext(fwder, client)
	}
}

func (fwder *Forwarder) processRequest(client *core.Client) {
	f := fwder.viewFwder.GetFwder(client.View, client.Request.Question.Name)
	if f == nil {
		logger.GetLogger().Debug("no zone fwder is specified for query %s in view %s", client.Request.Question.String(), client.View)
	} else {
		if err := f.SetQuerySource(querysource.GetQuerySource(client.View)); err != nil {
			logger.GetLogger().Error("view fwder failed:" + err.Error())
		} else {
			if resp, _, err := f.Forward(client.Request); err == nil {
				logger.GetLogger().Debug("send query %s to fwder %s succeed", client.Request.Question.String(), f.RemoteAddr())
				client.Response = resp
			} else {
				logger.GetLogger().Error("send query %s to fwder %s failed: %s", client.Request.Question.String(), f.RemoteAddr(), err.Error())
			}
		}
	}
}
