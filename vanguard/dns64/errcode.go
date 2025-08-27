package dns64

import (
	"vanguard/httpcmd"
)

var (
	ErrBadDns64Prefix     = httpcmd.NewError(httpcmd.DNS64ErrCodeStart, "dns64 prefix should be a ipv6 addr")
	ErrInvalidDns64Prefix = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+1, "invalid dns64 prefix netmask")
	ErrNoSuffix           = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+2, "suffix is needed if netmask of prefix smaller than 96")
	ErrDns64Exists        = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+3, "DNS64 setting already exists")
	ErrNonExistDns64      = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+4, "operate non-exist DNS64 setting")
	ErrBadDns64Suffix     = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+5, "dns64 suffix should be a ipv6 addr")
	ErrInvalidDns64Suffix = httpcmd.NewError(httpcmd.DNS64ErrCodeStart+6, "invalid dns64 suffix")
)
