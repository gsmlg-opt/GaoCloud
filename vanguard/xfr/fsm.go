package xfr

import (
	"cement/fsm"
	"g53"
	"vanguard/logger"
	"vanguard/resolver/auth/zone"

	"errors"
)

var (
	errSerialNumberIsWrong        = errors.New("serial number should in sequence")
	errWrapperSOAIsNotInPair      = errors.New("serial number of wrapper soa should in pair")
	errSOAShouldHasOneRR          = errors.New("soa rrset in axfr should has one rr")
	errIXFRDataSectionFormatError = errors.New("ixfr data section format isn't valid")
)

type XFRSMGenertor struct {
	typ           xfrType
	latestSerial  uint32
	currentSerial uint32
	updator       zone.ZoneUpdator
	tx            zone.Transaction
}

type XFRStateMachine struct {
	*fsm.FSM
	typ xfrType
}

func newFSMGenerator(typ xfrType, currentSerial, latestSerial uint32, updator zone.ZoneUpdator, tx zone.Transaction) *XFRSMGenertor {
	return &XFRSMGenertor{
		typ:           typ,
		updator:       updator,
		tx:            tx,
		currentSerial: currentSerial,
		latestSerial:  latestSerial,
	}
}

func (g *XFRSMGenertor) GenStateMachine() *XFRStateMachine {
	if g.typ == IXFR {
		return &XFRStateMachine{
			FSM: g.genIXFRStateMachine(),
			typ: IXFR,
		}
	} else {
		return &XFRStateMachine{
			FSM: g.genAXFRStateMachine(),
			typ: AXFR,
		}
	}
}

func (g *XFRSMGenertor) genIXFRStateMachine() *fsm.FSM {
	return fsm.NewFSM(
		"init",
		fsm.Events{
			{Name: "get_wrapper_soa", Src: []string{"init"}, Dst: "start"},
			{Name: "get_mid_soa", Src: []string{"start"}, Dst: "delete_section"},
			{Name: "get_mid_soa", Src: []string{"delete_section"}, Dst: "add_section"},
			{Name: "get_mid_soa", Src: []string{"add_section"}, Dst: "delete_section"},
			{Name: "get_wrapper_soa", Src: []string{"add_section"}, Dst: "done"},
			{Name: "get_none_soa", Src: []string{"delete_section"}, Dst: "delete_section"},
			{Name: "get_none_soa", Src: []string{"add_section"}, Dst: "add_section"},
		},
		fsm.Callbacks{
			"before_get_wrapper_soa": func(e *fsm.Event) {
				serial := e.Args[0].(uint32)
				if e.Src == "init" {
					g.latestSerial = serial
				} else if g.latestSerial != serial {
					e.Cancel(errWrapperSOAIsNotInPair)
				}
			},

			"before_get_mid_soa": func(e *fsm.Event) {
				serial := e.Args[0].(uint32)
				if e.Src == "start" || e.Src == "add_section" {
					if serial != g.currentSerial {
						e.Cancel(errSerialNumberIsWrong)
					}
				} else {
					if serial != g.currentSerial+1 {
						e.Cancel(errSerialNumberIsWrong)
					} else {
						g.updator.IncreaseSerialNumber(g.tx)
						g.currentSerial += 1
					}
				}
			},

			"after_get_none_soa": func(e *fsm.Event) {
				rrset := e.Args[0].(*g53.RRset)
				if e.Src == "delete_section" {
					g.updator.DeleteRRset(g.tx, rrset)
				} else {
					g.updator.Add(g.tx, rrset)
				}
			},
		},
	)
}

func (g *XFRSMGenertor) genAXFRStateMachine() *fsm.FSM {
	return fsm.NewFSM(
		"init",
		fsm.Events{
			{Name: "get_wrapper_soa", Src: []string{"init"}, Dst: "start"},
			{Name: "get_none_soa", Src: []string{"start"}, Dst: "add_section"},
			{Name: "get_none_soa", Src: []string{"add_section"}, Dst: "add_section"},
			{Name: "get_wrapper_soa", Src: []string{"add_section"}, Dst: "done"},
		},
		fsm.Callbacks{
			"before_get_wrapper_soa": func(e *fsm.Event) {
				soa := e.Args[0].(*g53.RRset)
				serial := soa.Rdatas[0].(*g53.SOA).Serial
				if serial != g.latestSerial {
					e.Cancel(errSerialNumberIsWrong)
				} else if e.Src == "init" {
					g.updator.Add(g.tx, soa)
				}
			},

			"after_get_none_soa": func(e *fsm.Event) {
				rrset := e.Args[0].(*g53.RRset)
				g.updator.Add(g.tx, rrset)
			},
		},
	)
}

func runIXFRFSM(sm *fsm.FSM, answers g53.Section) error {
	rrsetCount := len(answers)
	for i, rrset := range answers {
		if rrset.Type == g53.RR_SOA {
			rdataCount := rrset.RRCount()
			for j, rdata := range rrset.Rdatas {
				serial := rdata.(*g53.SOA).Serial
				if (i == 0 && j == 0) || (i == rrsetCount-1 && j == rdataCount-1) {
					if err := sm.Event("get_wrapper_soa", serial); err != nil {
						logger.GetLogger().Error("xfr protocol error: %s", err.Error())
						return err
					}
				} else {
					if err := sm.Event("get_mid_soa", serial); err != nil {
						logger.GetLogger().Error("xfr protocol err: %s", err.Error())
						return err
					}
				}
			}
		} else {
			err := sm.Event("get_none_soa", rrset)
			if _, ok := err.(fsm.NoTransitionError); ok == false {
				logger.GetLogger().Error("xfr protocol error: %s", err.Error())
				return errIXFRDataSectionFormatError
			}
		}
	}
	return nil
}

func runAXFRFSM(sm *fsm.FSM, answers g53.Section) error {
	rrsetCount := len(answers)
	for i, rrset := range answers {
		var err error
		if rrset.Type == g53.RR_SOA {
			rdataCount := rrset.RRCount()
			if rdataCount != 1 {
				return errSOAShouldHasOneRR
			}
			if i == 0 || i == rrsetCount-1 {
				err = sm.Event("get_wrapper_soa", rrset)
			}
		} else {
			err = sm.Event("get_none_soa", rrset)
		}
		if err != nil {
			if _, ok := err.(fsm.NoTransitionError); ok == false {
				logger.GetLogger().Error("xfr protocol error: %s", err.Error())
				return err
			}
		}
	}

	return nil
}

func (sm *XFRStateMachine) Run(answers g53.Section) error {
	if sm.typ == IXFR {
		return runIXFRFSM(sm.FSM, answers)
	} else {
		return runAXFRFSM(sm.FSM, answers)
	}
}
