package hijack

import (
	"strings"

	"vanguard/httpcmd"
	ld "vanguard/localdata"
)

type AddRedirectRR struct {
	View  string `json:"view"`
	Name  string `json:"name"`
	Ttl   string `json:"ttl"`
	Type  string `json:"type"`
	Rdata string `json:"rdata"`
}

func (r *AddRedirectRR) String() string {
	return "name: add redirect rr and params:{view:" + r.View +
		", name:" + r.Name +
		", ttl:" + r.Ttl +
		", type:" + r.Type +
		", rdata:" + r.Rdata + "}"
}

type DeleteRedirectRR struct {
	View  string `json:"view"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Rdata string `json:"rdata"`
}

func (r *DeleteRedirectRR) String() string {
	return "name: delete redirect rr and params:{view:" + r.View +
		", name:" + r.Name +
		", type:" + r.Type +
		", rdata:" + r.Rdata + "}"
}

type UpdateRedirectRR struct {
	View     string `json:"view"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	OldRdata string `json:"old_rdata"`
	NewTtl   string `json:"new_ttl"`
	NewRdata string `json:"new_rdata"`
}

func (r *UpdateRedirectRR) String() string {
	return "name: update redirect rr and params:{\ndelete rr: view:" + r.View +
		", name:" + r.Name +
		", type:" + r.Type +
		", rdata:" + r.OldRdata +
		"\nadd rr: view:" + r.View +
		", name:" + r.Name +
		", ttl:" + r.NewTtl +
		", type:" + r.Type +
		", rdata:" + r.NewRdata + "}"
}

func (h *Hijack) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddRedirectRR:
		return nil, h.rrsets.AddPolicies(c.View, ld.LPLocalRRset, []string{strings.Join([]string{c.Name, c.Ttl, c.Type, c.Rdata}, " ")})
	case *DeleteRedirectRR:
		return nil, h.rrsets.RemovePolicies(c.View, ld.LPLocalRRset, []string{strings.Join([]string{c.Name, "0", c.Type, c.Rdata}, " ")})
	case *UpdateRedirectRR:
		h.rrsets.RemovePolicies(c.View, ld.LPLocalRRset, []string{strings.Join([]string{c.Name, "0", c.Type, c.OldRdata}, " ")})
		return nil, h.rrsets.AddPolicies(c.View, ld.LPLocalRRset, []string{strings.Join([]string{c.Name, c.NewTtl, c.Type, c.NewRdata}, " ")})
	default:
		panic("should not be here")
	}
}
