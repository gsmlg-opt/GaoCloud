package forwarder

import (
	"sync"

	"g53"
	"vanguard/config"
	"vanguard/httpcmd"
	view "vanguard/viewselector"
)

const (
	FwderFixedOrderPolicy = "fixed_order"
	FwderRttPolicy        = "rtt"
	FwderRoundRobinPolicy = "round_robin"
	FwderMatchException   = "no"
)

var strToFwdSelectPolicy = map[string]FwdSelectPolicy{
	FwderFixedOrderPolicy: fixedOrder,
	FwderRttPolicy:        rttBased,
	FwderRoundRobinPolicy: roundRobin,
}

type ViewFwderMgr struct {
	fwders map[string]*ViewFwder
	repo   *SafeFwderRepo
	lock   sync.RWMutex
}

func NewViewFwderMgr(conf *config.VanguardConf) *ViewFwderMgr {
	mgr := &ViewFwderMgr{}
	mgr.ReloadConfig(conf)
	httpcmd.RegisterHandler(mgr, []httpcmd.Command{&AddForwardZone{}, &DeleteForwardZone{}, &UpdateForwardZone{}})
	return mgr
}

func (mgr *ViewFwderMgr) ReloadConfig(conf *config.VanguardConf) {
	if mgr.repo == nil {
		mgr.repo = NewSafeFwderRepo(&conf.Forwarder.Prober)
	} else {
		mgr.repo.ReloadConf(&conf.Forwarder.Prober)
	}

	viewFwders := make(map[string]*ViewFwder)
	for view, _ := range view.GetViewAndIds() {
		viewFwders[view] = newViewFwder()
	}

	for _, c := range conf.Forwarder.ForwardZones {
		viewFwder := viewFwders[c.View]
		for _, zone := range c.Zones {
			zoneFwder, err := mgr.newZoneForwarder(zone.Name, zone.ForwardStyle, zone.Forwarders)
			if err != nil {
				panic("load forward zone " + zone.Name + " failed:" + err.Error())
			}

			if err := viewFwder.addZoneFwder(zone.Name, zoneFwder); err != nil {
				panic("load forward zone " + zone.Name + " failed:" + err.Error())
			}
		}
	}
	mgr.fwders = viewFwders
}

func (mgr *ViewFwderMgr) newZoneForwarder(name, style string, forwarders []string) (*ZoneFwder, error) {
	matchType := matchSubdomain
	policy := roundRobin
	if style == FwderMatchException {
		matchType = matchException
	} else {
		policy = strToFwdSelectPolicy[style]
	}

	fwders := []SafeFwder{}
	for _, addr := range forwarders {
		if forwarder, err := mgr.repo.GetOrCreateFwder(addr); err == nil {
			fwders = append(fwders, forwarder)
		} else {
			return nil, err
		}
	}

	var zoneFwder *ZoneFwder
	if len(fwders) == 1 {
		zoneFwder = newZoneFwder(matchType, fwders[0])
	} else {
		zoneFwder = newZoneFwder(matchType, NewFwderGroup(CreateSelector(policy, fwders)))
	}

	return zoneFwder, nil
}

func (mgr *ViewFwderMgr) GetFwder(view string, name *g53.Name) SafeFwder {
	mgr.lock.RLock()
	viewFwder, ok := mgr.fwders[view]
	mgr.lock.RUnlock()
	if ok {
		return viewFwder.getFwder(name)
	} else {
		return nil
	}
}
