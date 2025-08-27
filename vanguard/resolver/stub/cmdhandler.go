package stub

import (
	"strings"

	"g53"
	"vanguard/httpcmd"
)

type AddStubZone struct {
	View    string   `json:"view"`
	Name    string   `json:"name"`
	Masters []string `json:"masters"`
}

func (z *AddStubZone) String() string {
	return "name: add stub zone and params: {name:" + z.Name +
		", view:" + z.View +
		", masters:[" + strings.Join(z.Masters, ",") + "}"
}

type DeleteStubZone struct {
	View string `json:"view"`
	Name string `json:"name"`
}

func (z *DeleteStubZone) String() string {
	return "name: delete stub zone and params: {name:" + z.Name +
		", view:" + z.View + "}"
}

type UpdateStubZone struct {
	View    string   `json:"view"`
	Name    string   `json:"name"`
	Masters []string `json:"masters"`
}

func (z *UpdateStubZone) String() string {
	return "name: update stub zone and params: {name:" + z.Name +
		", view:" + z.View +
		", masters:[" + strings.Join(z.Masters, ",") + "}"
}

func (z *StubZoneManager) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddStubZone:
		return z.addStubZone(c.View, c.Name, c.Masters)
	case *DeleteStubZone:
		return z.deleteStubZone(c.View, c.Name)
	case *UpdateStubZone:
		return z.updateStubZone(c.View, c.Name, c.Masters)
	default:
		panic("should not be here")
	}
}

func (z *StubZoneManager) addStubZone(viewName, zoneName string, masters []string) (interface{}, *httpcmd.Error) {
	origin, err := g53.NameFromString(zoneName)
	if err != nil {
		return nil, httpcmd.ErrInvalidName.AddDetail(err.Error())
	}

	zones, ok := z.stubZones[viewName]
	if ok == false {
		return nil, httpcmd.ErrUnknownView.AddDetail(viewName)
	}

	z.lock.Lock()
	defer z.lock.Unlock()
	if _, err = zones.Insert(origin, masters); err != nil {
		return nil, ErrAddStubZoneFailed.AddDetail(err.Error())
	} else {
		return nil, nil
	}
}

func (z *StubZoneManager) deleteStubZone(viewName, zoneName string) (interface{}, *httpcmd.Error) {
	origin, err := g53.NameFromString(zoneName)
	if err != nil {
		return nil, httpcmd.ErrInvalidName.AddDetail(err.Error())
	}

	z.lock.Lock()
	z.stubZones[viewName].Delete(origin)
	z.lock.Unlock()

	return nil, nil
}

func (z *StubZoneManager) updateStubZone(viewName, zoneName string, masters []string) (interface{}, *httpcmd.Error) {
	origin, err := g53.NameFromString(zoneName)
	if err != nil {
		return nil, httpcmd.ErrInvalidName.AddDetail(err.Error())
	}

	z.lock.Lock()
	defer z.lock.Unlock()
	z.stubZones[viewName].Delete(origin)

	if _, err = z.stubZones[viewName].Insert(origin, masters); err != nil {
		return nil, ErrUpdateStubZoneFailed.AddDetail(err.Error())
	} else {
		return nil, nil
	}
}
