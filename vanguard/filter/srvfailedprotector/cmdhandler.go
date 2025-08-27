package srvfailedprotector

import (
	"sync/atomic"

	"vanguard/httpcmd"
)

type SrvFailedProtect struct {
	Enable bool `json:"enable"`
}

func (r *SrvFailedProtect) String() string {
	if r.Enable {
		return "enable server failed protect"
	} else {
		return "disable server failed protect"
	}
}

func (p *SFProtector) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *SrvFailedProtect:
		if c.Enable {
			atomic.StoreInt32(&p.drop, 1)
		} else {
			atomic.StoreInt32(&p.drop, 0)
		}
		return nil, nil
	default:
		panic("should not be here")
	}
}
