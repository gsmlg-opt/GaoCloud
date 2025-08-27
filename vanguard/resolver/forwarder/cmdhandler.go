package forwarder

import (
	"strings"

	"vanguard/httpcmd"
)

type ForwardZoneParam struct {
	View         string   `json:"view"`
	Name         string   `json:"name"`
	Forwarders   []string `json:"forwarders"`
	ForwardStyle string   `json:"forward_style"`
}

type AddForwardZone struct {
	Zones []ForwardZoneParam `json:"zones"`
}

func (f *AddForwardZone) String() string {
	var desc string
	for _, z := range f.Zones {
		desc += "name: add forward zone and params:{view:" + z.View +
			", name:" + z.Name +
			", forwarders:[" + strings.Join(z.Forwarders, ",") +
			"], forward_style:" + z.ForwardStyle + "},"
	}
	return desc
}

type DeleteForwardZone struct {
	View string `json:"view"`
	Name string `json:"name"`
}

func (f *DeleteForwardZone) String() string {
	return "name: delete forward zone and params:{view:" + f.View + ", name: " + f.Name + "}"
}

type UpdateForwardZone struct {
	View         string   `json:"view"`
	Name         string   `json:"name"`
	Forwarders   []string `json:"forwarders"`
	ForwardStyle string   `json:"forward_style"`
}

func (f *UpdateForwardZone) String() string {
	return "name: update forward zone and params:{view:" + f.View +
		", name:" + f.Name +
		", forwarders:[" + strings.Join(f.Forwarders, ",") +
		"], forward_style:" + f.ForwardStyle + "}"
}

func (m *ViewFwderMgr) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddForwardZone:
		return nil, m.addForwardZone(c.Zones)
	case *DeleteForwardZone:
		return nil, m.deleteForwardZone(c.View, c.Name)
	case *UpdateForwardZone:
		return nil, m.updateForwardZone(c.View, c.Name, c.ForwardStyle, c.Forwarders)
	default:
		panic("should not be here")
	}
}

func (m *ViewFwderMgr) addForwardZone(zones []ForwardZoneParam) *httpcmd.Error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, z := range zones {
		viewFwder, ok := m.fwders[z.View]
		if ok == false {
			return httpcmd.ErrUnknownView.AddDetail(z.View)
		}

		zoneFwder, err := m.newZoneForwarder(z.Name, z.ForwardStyle, z.Forwarders)
		if err != nil {
			return ErrAddForwardZoneFailed.AddDetail(err.Error())
		}

		if err := viewFwder.addZoneFwder(z.Name, zoneFwder); err != nil {
			return ErrAddForwardZoneFailed.AddDetail(err.Error())
		}
	}
	return nil
}

func (m *ViewFwderMgr) deleteForwardZone(view, name string) *httpcmd.Error {
	viewFwder, ok := m.fwders[view]
	if ok == false {
		return httpcmd.ErrUnknownView.AddDetail(view)
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	if err := viewFwder.deleteZoneFwder(name); err != nil {
		return ErrDeleteForwardZoneFailed.AddDetail(err.Error())
	} else {
		return nil
	}
}

func (m *ViewFwderMgr) updateForwardZone(view, name, style string, forwarders []string) *httpcmd.Error {
	viewFwder, ok := m.fwders[view]
	if ok == false {
		return httpcmd.ErrUnknownView.AddDetail(view)
	}

	zoneFwder, err := m.newZoneForwarder(name, style, forwarders)
	if err != nil {
		return ErrUpdateForwardZoneFailed.AddDetail(err.Error())
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	if err := viewFwder.deleteZoneFwder(name); err != nil {
		return ErrUpdateForwardZoneFailed.AddDetail(err.Error())
	}

	if err := viewFwder.addZoneFwder(name, zoneFwder); err != nil {
		return ErrUpdateForwardZoneFailed.AddDetail(err.Error())
	}

	return nil
}
