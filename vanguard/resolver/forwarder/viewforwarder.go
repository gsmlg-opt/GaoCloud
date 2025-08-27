package forwarder

import (
	"cement/domaintree"
	"g53"
)

type ZoneMatchType int

const (
	matchSubdomain ZoneMatchType = 0 + iota
	matchExact
	matchException
)

type ZoneFwder struct {
	matchType  ZoneMatchType
	fwderGroup SafeFwder
}

func newZoneFwder(matchType ZoneMatchType, fwderGroup SafeFwder) *ZoneFwder {
	return &ZoneFwder{
		fwderGroup: fwderGroup,
		matchType:  matchType,
	}
}

type ViewFwder struct {
	zoneFwders *domaintree.DomainTree //tree of zoneForwarder
}

func newViewFwder() *ViewFwder {
	return &ViewFwder{
		zoneFwders: domaintree.NewDomainTree(),
	}
}

func (f *ViewFwder) getFwder(name *g53.Name) SafeFwder {
	parents, match := f.zoneFwders.SearchParents(name)
	if match == domaintree.NotFound {
		return nil
	}

	for parents.IsEmpty() == false {
		zoneFwder := parents.Top().Data().(*ZoneFwder)
		switch zoneFwder.matchType {
		case matchException:
			return nil
		case matchSubdomain:
			return zoneFwder.fwderGroup
		case matchExact:
			if match == domaintree.ExactMatch {
				return zoneFwder.fwderGroup
			} else {
				parents.Pop()
			}
		}
	}
	return nil
}

func (f *ViewFwder) addZoneFwder(name string, forwarder *ZoneFwder) error {
	dname, err := g53.NameFromString(name)
	if err != nil {
		return err
	}

	_, err = f.zoneFwders.Insert(dname, forwarder)
	if err != nil {
		return ErrDuplicateForwardZone
	} else {
		return nil
	}
}

func (f *ViewFwder) deleteZoneFwder(name string) error {
	dname, err := g53.NameFromString(name)
	if err != nil {
		return err
	}

	f.zoneFwders.Delete(dname)
	return nil
}
