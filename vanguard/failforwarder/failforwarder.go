package failforwarder

import (
	"sync"
	"time"

	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	"vanguard/logger"
	"vanguard/resolver/querysource"
	"vanguard/util"
)

var (
	viewNameForDefaultForwarder = "*"
	defaultTimeout              = 2 * time.Second
)

type forwarder struct {
	sender *util.SafeUDPSender
	server string
}

type FailForwarder struct {
	core.DefaultHandler
	forwarders map[string]*forwarder
	lock       sync.RWMutex
}

func NewFailForwarder(conf *config.VanguardConf) core.DNSQueryHandler {
	ff := &FailForwarder{}
	ff.ReloadConfig(conf)
	httpcmd.RegisterHandler(ff, []httpcmd.Command{&AddFailForwarder{}, &DeleteFailForwarder{}, &UpdateFailForwarder{}})
	return ff
}

func (ff *FailForwarder) ReloadConfig(conf *config.VanguardConf) {
	fs := make(map[string]*forwarder)
	for _, c := range conf.FailForwarder {
		f, err := util.NewSafeUDPSender(querysource.GetQuerySource(c.View), defaultTimeout)
		if err != nil {
			panic("valid query source:" + err.Error())
		}
		fs[c.View] = &forwarder{
			sender: f,
			server: c.Forwarder,
		}
	}
	ff.forwarders = fs
}

func (ff *FailForwarder) HandleQuery(ctx *core.Context) {
	client := &ctx.Client
	if f := ff.GetForwarder(client.View); f != nil {
		response, _, err := f.sender.Query(f.server, client.Request)
		if err == nil {
			client.Response = response
		} else {
			logger.GetLogger().Error("fail forwarder failed:%s", err.Error())
		}
	} else {
		core.PassToNext(ff, ctx)
	}
}

func (ff *FailForwarder) GetForwarder(view string) *forwarder {
	ff.lock.RLock()
	defer ff.lock.RUnlock()
	if f, ok := ff.forwarders[view]; ok {
		return f
	} else if f, ok := ff.forwarders[viewNameForDefaultForwarder]; ok {
		return f
	} else {
		return nil
	}
}

func (ff *FailForwarder) AddForwarder(view, server string) *httpcmd.Error {
	f, err := util.NewSafeUDPSender(querysource.GetQuerySource(view), defaultTimeout)
	if err != nil {
		return ErrInvalidFailForwarder.AddDetail(err.Error())
	}

	ff.lock.Lock()
	defer ff.lock.Unlock()
	if _, ok := ff.forwarders[view]; ok {
		return ErrDuplicateFailForwarder
	}

	ff.forwarders[view] = &forwarder{
		sender: f,
		server: server,
	}
	return nil
}

func (ff *FailForwarder) DeleteForwarder(view string) *httpcmd.Error {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	if _, ok := ff.forwarders[view]; ok == false {
		return ErrNotExistFailForwarder
	} else {
		delete(ff.forwarders, view)
		return nil
	}
}

func (ff *FailForwarder) UpdateForwarder(view, server string) *httpcmd.Error {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	f, ok := ff.forwarders[view]
	if ok == false {
		return ErrNotExistFailForwarder
	}
	f.server = server
	return nil
}
