package fakeauth

import (
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	ld "vanguard/localdata"
	"vanguard/logger"
	"vanguard/resolver/chain"
)

type FakeAuth struct {
	chain.DefaultResolver
	localdata *ld.LocalData
}

func NewFakeAuth(conf *config.VanguardConf) *FakeAuth {
	f := &FakeAuth{}
	f.ReloadConfig(conf)
	httpcmd.RegisterHandler(f, []httpcmd.Command{&AddLocalData{}, &DeleteLocalData{}, &UpdateLocalData{}})
	return f
}

func (f *FakeAuth) ReloadConfig(conf *config.VanguardConf) {
	f.localdata = ld.NewLocalData()
	for _, c := range conf.LocalData {
		if err := f.localdata.AddPolicies(c.View, ld.LPNXDomain, c.NXDomain); err != nil {
			panic("invalid nxdomain:" + err.Error())
		}
		if err := f.localdata.AddPolicies(c.View, ld.LPNXRRset, c.NXRRset); err != nil {
			panic("invalid nxrrset:" + err.Error())
		}
		if err := f.localdata.AddPolicies(c.View, ld.LPExceptionDomain, c.Exception); err != nil {
			panic("invalid exception:" + err.Error())
		}
		if err := f.localdata.AddPolicies(c.View, ld.LPLocalRRset, c.Redirect); err != nil {
			panic("invalid redirect:" + err.Error())
		}
	}
}

func (l *FakeAuth) Resolve(client *core.Client) {
	if l.localdata.ResponseWithLocalData(client) {
		logger.GetLogger().Debug("found rrset for name %s in view %s in local data",
			client.Request.Question.Name.String(false), client.View)
		client.CacheAnswer = false
	} else {
		chain.PassToNext(l, client)
	}
}
