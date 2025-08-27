package ratelimit

import (
	"fmt"
	"net"
	"strings"

	"cement/domaintree"
	"g53"
	"vanguard/httpcmd"
)

type AddNameRateLimit struct {
	View      string `json:"view"`
	Name      string `json:"name"`
	RateLimit uint32 `json:"rate_limit"`
}

func (r *AddNameRateLimit) String() string {
	return fmt.Sprintf("name: add name rrls and params:{view:%v, name:%v, rate_limit:%v}",
		r.View, r.Name, r.RateLimit)
}

type DeleteNameRateLimit struct {
	View string `json:"view"`
	Name string `json:"name"`
}

func (r *DeleteNameRateLimit) String() string {
	return fmt.Sprintf("name: add name rrls and params:{view:%v, name:%v}",
		r.View, r.Name)
}

type UpdateNameRateLimit struct {
	View      string `json:"view"`
	Name      string `json:"name"`
	RateLimit uint32 `json:"rate_limit"`
}

func (r *UpdateNameRateLimit) String() string {
	return fmt.Sprintf("name: update name rrls and params:{view:%v, name:%v, rate_limit:%v}",
		r.View, r.Name, r.RateLimit)
}

func (t *NameThrottler) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddNameRateLimit:
		return t.addNameRateLimit(c.View, c.Name, c.RateLimit)
	case *DeleteNameRateLimit:
		return t.deleteNameRateLimit(c.View, c.Name)
	case *UpdateNameRateLimit:
		return t.updateNameRateLimit(c.View, c.Name, c.RateLimit)
	default:
		panic("should not br here")
	}
}

func (t *NameThrottler) addNameRateLimit(view, name string, limit uint32) (interface{}, *httpcmd.Error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	records, found := t.viewRecords[view]
	if found == false {
		records = domaintree.NewDomainTree()
		t.viewRecords[view] = records
	}

	return nil, doAddNameRateLimit(records, name, limit)
}

func doAddNameRateLimit(tree *domaintree.DomainTree, name string, limit uint32) *httpcmd.Error {
	origin, matchType, err := parseName(name)
	if err != nil {
		return err
	}

	if _, err_ := tree.Insert(origin, newNameAccessRecord(matchType, limit)); err_ != nil {
		return ErrAddNameRateLimitFailed.AddDetail(err_.Error())
	}

	return nil
}

func (t *NameThrottler) deleteNameRateLimit(view, name string) (interface{}, *httpcmd.Error) {
	origin, _, err := parseName(name)
	if err != nil {
		return nil, httpcmd.ErrInvalidName.AddDetail(err.Error())
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	records, found := t.viewRecords[view]
	if found == false {
		return nil, httpcmd.ErrUnknownView.AddDetail(view)
	}

	records.Delete(origin)
	return nil, nil
}

func (t *NameThrottler) updateNameRateLimit(view, name string, ratelimit uint32) (interface{}, *httpcmd.Error) {
	origin, matchType, err := parseName(name)
	if err != nil {
		return nil, httpcmd.ErrInvalidName.AddDetail(err.Error())
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	records, found := t.viewRecords[view]
	if found == false {
		return nil, httpcmd.ErrUnknownView.AddDetail(view)
	}

	records.Delete(origin)
	if _, err := records.Insert(origin, newNameAccessRecord(matchType, ratelimit)); err != nil {
		return nil, ErrUpdateNameRateLimitFailed.AddDetail(err.Error())
	}

	return nil, nil
}

func parseName(name string) (*g53.Name, NameMatchType, *httpcmd.Error) {
	matchType := exactMatch
	if strings.HasPrefix(name, "*.") {
		name = name[2:]
		matchType = zoneMatch
	}

	origin, err := g53.NameFromString(name)
	if err != nil {
		return nil, matchType, httpcmd.ErrInvalidName.AddDetail(err.Error())
	} else {
		return origin, matchType, nil
	}
}

type AddIpRateLimit struct {
	Network   string `json:"network"`
	RateLimit uint32 `json:"rate_limit"`
}

func (r *AddIpRateLimit) String() string {
	return fmt.Sprintf("name: add ip rrls and params:{network:%v, rate_limit:%v}", r.Network, r.RateLimit)
}

type DeleteIpRateLimit struct {
	Network string `json:"network"`
}

func (r *DeleteIpRateLimit) String() string {
	return fmt.Sprintf("name: delete ip rrls and params:{network:%v}", r.Network)
}

type UpdateIpRateLimit struct {
	Network   string `json:"network"`
	RateLimit uint32 `json:"rate_limit"`
}

func (r *UpdateIpRateLimit) String() string {
	return fmt.Sprintf("name: update ip rrls and params:{network:%v, rate_limit:%v}", r.Network, r.RateLimit)
}

func (t *IPThrottler) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddIpRateLimit:
		return t.addIpRateLimit(c.Network, c.RateLimit)
	case *DeleteIpRateLimit:
		return t.deleteIpRateLimit(c.Network)
	case *UpdateIpRateLimit:
		return t.updateIpRateLimit(c.Network, c.RateLimit)
	default:
		panic("should not be here")
	}
}

func (t *IPThrottler) addIpRateLimit(network string, ratelimit uint32) (interface{}, *httpcmd.Error) {
	_, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, httpcmd.ErrInvalidNetwork.AddDetail(err.Error())
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	if err := t.entry.networks.Add(ipnet.String(), ratelimit); err != nil {
		return nil, ErrAddIPRateLimitFailed.AddDetail(err.Error())
	} else {
		return nil, nil
	}
}

func (t *IPThrottler) deleteIpRateLimit(network string) (interface{}, *httpcmd.Error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if err := t.entry.networks.Delete(network); err != nil {
		return nil, ErrDeleteIPRateLimitFailed.AddDetail(err.Error())
	} else {
		return nil, nil
	}
}

func (t *IPThrottler) updateIpRateLimit(network string, ratelimit uint32) (interface{}, *httpcmd.Error) {
	_, _, err := net.ParseCIDR(network)
	if err != nil {
		return nil, httpcmd.ErrInvalidNetwork.AddDetail(err.Error())
	}

	t.lock.Lock()
	defer t.lock.Unlock()
	if err := t.entry.networks.Delete(network); err != nil {
		return nil, ErrUpdateIPRateLimitFailed.AddDetail(err.Error())
	}

	if err := t.entry.networks.Add(network, ratelimit); err != nil {
		return nil, ErrUpdateIPRateLimitFailed.AddDetail(err.Error())
	}

	return nil, nil
}
