package sortlist

import (
	"strings"

	"vanguard/httpcmd"
)

type AddSortList struct {
	View        string   `json:"view"`
	SourceIp    string   `json:"source_ip"`
	PreferedIps []string `json:"prefered_ips"`
}

func (s *AddSortList) String() string {
	return "name: add sortlist and params: {view:" + s.View +
		", source_ip:" + s.SourceIp +
		", prefered_ips:[" + strings.Join(s.PreferedIps, ",") + "]}"
}

type DeleteSortList struct {
	View     string `json:"view"`
	SourceIp string `json:"source_ip"`
}

func (s *DeleteSortList) String() string {
	return "name: delete sortlist and params: {view:" + s.View +
		", source_ip:" + s.SourceIp + "}"
}

type UpdateSortList struct {
	View        string   `json:"view"`
	SourceIp    string   `json:"source_ip"`
	PreferedIps []string `json:"prefered_ips"`
}

func (s *UpdateSortList) String() string {
	return "name: update sortlist and params: {view:" + s.View +
		", source_ip:" + s.SourceIp +
		", prefered_ips:[" + strings.Join(s.PreferedIps, ",") + "]}"
}

func (s *UserAddrBasedSorter) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddSortList:
		return nil, s.addSortList(c.View, c.SourceIp, c.PreferedIps)
	case *DeleteSortList:
		return nil, s.deleteSortList(c.View, c.SourceIp)
	case *UpdateSortList:
		return nil, s.updateSortList(c.View, c.SourceIp, c.PreferedIps)
	default:
		panic("should not be here")
	}
}

func (s *UserAddrBasedSorter) addSortList(view, sourceIp string, preferedIps []string) *httpcmd.Error {
	if err := s.addSorter(view, sourceIp, preferedIps); err != nil {
		return ErrAddSortListFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (s *UserAddrBasedSorter) deleteSortList(view, sourceIp string) *httpcmd.Error {
	if err := s.deleteSorter(view, sourceIp); err != nil {
		return ErrDeleteSortListFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (s *UserAddrBasedSorter) updateSortList(view, sourceIp string, preferedIps []string) *httpcmd.Error {
	if err := s.updateSorter(view, sourceIp, preferedIps); err != nil {
		return ErrDeleteSortListFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}
