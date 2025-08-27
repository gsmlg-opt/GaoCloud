package xfr

import (
	"sync"

	"cement/domaintree"
	"g53"
	"g53/util"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/auth"
	"vanguard/resolver/auth/zone"
)

type xfrType string

const (
	AXFR xfrType = "axfr"
	IXFR xfrType = "ixfr"
)

type XFRRunner struct {
	auth *auth.AuthDataSource

	mu           sync.Mutex
	inFlightIXFR map[string][]*g53.Name
}

func newXFRRunner(auth *auth.AuthDataSource) *XFRRunner {
	return &XFRRunner{
		auth:         auth,
		inFlightIXFR: make(map[string][]*g53.Name),
	}
}

func (h *XFRRunner) HandleNotify(ctx *core.Context) {
	notify := ctx.Client.Request
	answers := notify.Sections[g53.AnswerSection]
	if len(answers) != 1 || answers[0].Type != g53.RR_SOA {
		logger.GetLogger().Error("invalid notify has no soa rrset in answer section")
		return
	}

	targetZoneName := answers[0].Name
	view := ctx.Client.View
	targetZone, matchType := h.auth.GetZone(view, targetZoneName)
	logger.GetLogger().Debug("get notify for zone: %s in view: %s", targetZoneName.String(false), view)
	if matchType != domaintree.ExactMatch {
		logger.GetLogger().Warn("get notify with unknown zone: %s in view: %s", targetZoneName.String(false), view)
		return
	}

	if h.addZoneToTransfer(view, targetZoneName) == false {
		logger.GetLogger().Warn("zone: %s in view: %s is under ixfr", targetZoneName.String(false), view)
		//let the master, notify the slave later
		return
	}

	latestSerial := answers[0].Rdatas[0].(*g53.SOA).Serial
	if targetZone.IsMaster() {
		logger.GetLogger().Warn("zone: %s in view: %s is master", targetZoneName.String(false), view)
		return
	}

	master := ctx.Client.IP().String() + ":53"
	isKnownMaster := false
	for _, m := range targetZone.Masters() {
		if m == master {
			isKnownMaster = true
			break
		}
	}
	if isKnownMaster == false {
		logger.GetLogger().Warn("zone: %s in view: %s get notified from unknown master: %s", targetZoneName.String(false), view, master)
		return
	}

	ctx.Client.Response = notify.MakeResponse()
	go h.doXFR(view, targetZone, latestSerial, master)
}

func (h *XFRRunner) addZoneToTransfer(view string, zone *g53.Name) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	zones, ok := h.inFlightIXFR[view]
	if ok {
		for _, z := range zones {
			if z.Equals(zone) {
				logger.GetLogger().Warn("zone: %s in view: %s is run ixfr now", zone.String(false), view)
				return false
			}
		}
	}

	h.inFlightIXFR[view] = append(zones, zone)
	return true
}

func (h *XFRRunner) removeZoneFromTransfer(view string, zone *g53.Name) {
	h.mu.Lock()
	defer h.mu.Unlock()

	zones, _ := h.inFlightIXFR[view]
	for i, z := range zones {
		if z.Equals(zone) {
			zones = append(zones[:i], zones[i+1:]...)
			break
		}
	}
	h.inFlightIXFR[view] = zones
}

func (h *XFRRunner) doXFR(view string, z zone.Zone, latestSerial uint32, master string) {
	name := z.GetOrigin()
	defer h.removeZoneFromTransfer(view, name)

	var request *g53.Message
	var currentSerial uint32
	var xfrType xfrType
	soa := z.Find(name, g53.RR_SOA, zone.DefaultFind).GetResult().RRset
	if soa == nil {
		logger.GetLogger().Info("get notify for zone: %s in view: %s and vanguard have not data and will do axfr", name.String(false), view)
		xfrType = AXFR
		request = g53.MakeAXFR(name, nil)
	} else {
		currentSerial = soa.Rdatas[0].(*g53.SOA).Serial
		if g53.CompareSerial(currentSerial, latestSerial) != -1 {
			logger.GetLogger().Warn("get notify for zone: %s in view: %s, which serial number is smaller than us", name.String(false), view)
			return
		}
		xfrType = IXFR
		logger.GetLogger().Info("get notify for zone: %s in view: %s and vanguard will do ixfr", name.String(false), view)
		request = g53.MakeIXFR(name, soa, nil)
	}

	conn, err := util.NewTCPConn(master)
	if err != nil {
		logger.GetLogger().Error("connect to server %s failed: %s", master, err.Error())
		return
	}

	render := g53.NewMsgRender()
	request.Rend(render)
	if err := util.TCPWrite(render.Data(), conn); err != nil {
		logger.GetLogger().Error("send ixfr quer to server %s failed: %s", master, err.Error())
		return
	}

	answerBuffer, err := util.TCPRead(conn)
	if err != nil {
		logger.GetLogger().Error("connect to server %s failed: %s", master, err.Error())
		return
	}

	resp, err := g53.MessageFromWire(util.NewInputBuffer(answerBuffer))
	answers := resp.Sections[g53.AnswerSection]
	if err != nil {
		logger.GetLogger().Error("invalid %s response: %v", xfrType, err)
	} else if len(answers) == 0 {
		logger.GetLogger().Error("empty %s response", xfrType)
	}

	h.updateZoneUseXFR(xfrType, z, currentSerial, latestSerial, answers)
}

func (h *XFRRunner) updateZoneUseXFR(typ xfrType, z zone.Zone, currentSerial, latestSerial uint32, answers g53.Section) {
	updator, _ := z.GetUpdator(nil, true)
	tx, err := updator.Begin()
	if err != nil {
		logger.GetLogger().Error("get zone transaction failed: %s", err.Error())
		return
	}

	sm := newFSMGenerator(typ, currentSerial, latestSerial, updator, tx).GenStateMachine()
	if err := sm.Run(answers); err == nil {
		tx.Commit()
		logger.GetLogger().Info("%s succeed", typ)
	} else {
		tx.RollBack()
		logger.GetLogger().Warn("%s failed: %s", typ, err.Error())
		if typ == IXFR {
			logger.GetLogger().Info("IXFR failed try AXFR")
			h.updateZoneUseXFR(AXFR, z, currentSerial, latestSerial, answers)
		}
	}
}
