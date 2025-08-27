package stub

import (
	"vanguard/httpcmd"
)

var (
	ErrAddStubZoneFailed    = httpcmd.NewError(httpcmd.StubZoneErrCodeStart, "add stub zone failed")
	ErrDeleteStubZoneFailed = httpcmd.NewError(httpcmd.StubZoneErrCodeStart+1, "delete stub zone failed")
	ErrUpdateStubZoneFailed = httpcmd.NewError(httpcmd.StubZoneErrCodeStart+2, "update stub zone failed")
)
