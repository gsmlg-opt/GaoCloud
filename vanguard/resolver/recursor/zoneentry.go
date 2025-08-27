package recursor

import (
	"bytes"
	"time"

	"g53"
)

type TrustLevel uint8

const (
	OutOfZone   TrustLevel = 0
	FromReferal TrustLevel = 1
	FromAuth    TrustLevel = 2
)

type ZoneEntry struct {
	zone        *g53.Name
	nameServers []*g53.Name
	expireTime  time.Time
	trustLevel  TrustLevel
}

func newZoneEntry(name *g53.Name, nameServers []*g53.Name, ttl time.Duration, trustLevel TrustLevel) *ZoneEntry {
	return &ZoneEntry{
		zone:        name,
		nameServers: nameServers,
		expireTime:  time.Now().Add(ttl),
		trustLevel:  trustLevel,
	}
}

func (zone *ZoneEntry) selectNameServer(nameServers *NameServerManager) (servers []*NameServer) {
	for _, name := range zone.nameServers {
		ns := nameServers.getNameServer(name)
		if ns == nil || ns.isExpired() {
			continue
		}

		server := ns.selectNameServer()
		server.zone = zone.zone
		servers = append(servers, server)
	}
	return
}

func (zone *ZoneEntry) isExpired() bool {
	return zone.expireTime.Before(time.Now())
}

func (zone *ZoneEntry) String() string {
	var buf bytes.Buffer
	buf.WriteString("zone ")
	buf.WriteString(zone.zone.String(true))
	buf.WriteString(" with servers:\n")
	for _, s := range zone.nameServers {
		buf.WriteString(s.String(true))
		buf.WriteString("\n")
	}
	return buf.String()
}
