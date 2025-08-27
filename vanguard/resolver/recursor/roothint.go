package recursor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"g53"
)

var (
	errInvalidRootHintZoneName = errors.New("root hint zone name should be root")
	errUnsupportRRType         = errors.New("only ns and a is supported in root hint")
)

func loadRootServer(content string) ([]*NameServer, error) {
	var nameServers []*NameServer
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r\n ")
		if line == "" {
			continue
		}

		if rrset, err := g53.RRsetFromString(line); err != nil {
			return nil, fmt.Errorf("invalid rr %s", err.Error())
		} else if rrset.Type == g53.RR_NS {
			if rrset.Name.Equals(g53.Root) == false {
				return nil, errInvalidRootHintZoneName
			}
		} else if rrset.Type == g53.RR_A {
			nameServers = append(nameServers, &NameServer{
				zone: g53.Root,
				name: rrset.Name,
				addr: rrset.Rdatas[0].String() + ":53",
				rtt:  time.Duration(rrset.Ttl) * time.Second,
			})
		} else {
			return nil, errUnsupportRRType
		}
	}
	return nameServers, nil
}
