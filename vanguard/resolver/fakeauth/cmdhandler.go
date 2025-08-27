package fakeauth

import (
	"vanguard/httpcmd"
	ld "vanguard/localdata"
)

type LocalPolicyData struct {
	Policy ld.LocalPolicyType `json:"policy"`
	View   string             `json:"view"`
	Data   string             `json:"data"`
}

func (l *LocalPolicyData) toString() string {
	return "type:" + string(l.Policy) +
		"view:" + l.View +
		", data:" + l.Data
}

type AddLocalData struct {
	Data *LocalPolicyData `json:"localdata"`
}

func (l *AddLocalData) String() string {
	return "name: add localdata and params:{" + l.Data.toString() + "}"
}

type DeleteLocalData struct {
	Data *LocalPolicyData `json:"localdata"`
}

func (l *DeleteLocalData) String() string {
	return "name: delete localdata and params:{" + l.Data.toString() + "}"
}

type UpdateLocalData struct {
	OldData *LocalPolicyData `json:"old_localdata"`
	NewData *LocalPolicyData `json:"new_localdata"`
}

func (l *UpdateLocalData) String() string {
	return "name: update localdata and params:{delete localdata:" + l.OldData.toString() +
		"\nadd localdata:" + l.NewData.toString()
}

func (f *FakeAuth) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddLocalData:
		return nil, f.localdata.AddPolicies(c.Data.View, c.Data.Policy, []string{c.Data.Data})
	case *DeleteLocalData:
		return nil, f.localdata.RemovePolicies(c.Data.View, c.Data.Policy, []string{c.Data.Data})
	case *UpdateLocalData:
		f.localdata.RemovePolicies(c.OldData.View, c.OldData.Policy, []string{c.OldData.Data})
		return nil, f.localdata.AddPolicies(c.NewData.View, c.NewData.Policy, []string{c.NewData.Data})
	default:
		panic("should not be here")
	}
}
