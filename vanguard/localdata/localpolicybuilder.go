package localdata

import (
	"errors"
	"strconv"
	"strings"

	"vanguard/util"
)

type LocalPolicyType string

const (
	LPNXDomain        LocalPolicyType = "nxdomain"
	LPNXRRset                         = "nodata"
	LPExceptionDomain                 = "passthru"
	LPLocalRRset                      = "redirect"
)

var (
	ErrRedirectRRInvalid = errors.New("redirect rr should in format name, type, ttl, rdata")
)

func BuildLocalPolicy(lPType LocalPolicyType, data string) (LocalPolicy, error) {
	fields := strings.Split(data, " ")
	name := fields[0]
	isZoneMatch, origin, err := util.NameStripFirstWildcard(name)
	if err != nil {
		return nil, err
	}

	base := &basePolicy{
		isZoneMatch: isZoneMatch,
		name:        origin,
	}

	switch lPType {
	case LPNXDomain:
		return &NXDomain{base}, nil
	case LPNXRRset:
		return &NXRRset{base}, nil
	case LPExceptionDomain:
		return &ExceptionDomain{base}, nil
	case LPLocalRRset:
		if len(fields) < 4 {
			return nil, ErrRedirectRRInvalid
		}
		if ttl, err := strconv.Atoi(fields[1]); err != nil {
			return nil, err
		} else {
			return newLocalRRset(base, fields[2], ttl, strings.Join(fields[3:], " "))
		}
	default:
		panic("unknown policy type")
	}
}
