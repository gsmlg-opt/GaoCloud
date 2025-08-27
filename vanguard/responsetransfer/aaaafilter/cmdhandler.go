package aaaafilter

import (
	"strings"

	"vanguard/httpcmd"
)

type PutAAAAFilter struct {
	View string   `json:"view"`
	Acls []string `json:"filter_aaaa_ips"`
}

func (c *PutAAAAFilter) String() string {
	return "name: update aaaa filter and params: {view: " + c.View +
		", filter_aaaa_ips:[" + strings.Join(c.Acls, ",") + "]}"
}

func (f *aaaaFilter) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *PutAAAAFilter:
		f.updateViewAcls(c.View, c.Acls)
		return nil, nil
	default:
		panic("should not be here")
	}
}

func (f *aaaaFilter) updateViewAcls(view string, acls []string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	if len(acls) == 0 {
		delete(f.viewAndAcls, view)
	} else {
		f.viewAndAcls[view] = acls
	}
	return nil
}
