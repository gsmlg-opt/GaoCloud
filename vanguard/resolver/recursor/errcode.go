package recursor

import (
	"vanguard/httpcmd"
)

var (
	ErrRootZoneConflict    = httpcmd.NewError(httpcmd.RecursorErrCodeStart, "already exists root zone")
	ErrLimitedRrHint       = httpcmd.NewError(httpcmd.RecursorErrCodeStart+1, "only A and NS is allowed in root zone")
	ErrHintZoneExist       = httpcmd.NewError(httpcmd.RecursorErrCodeStart+2, "already has root configuration")
	ErrRootZoneNameInvalid = httpcmd.NewError(httpcmd.RecursorErrCodeStart+3, "root zone NS name must be (.) ")
	ErrNonExistHintZone    = httpcmd.NewError(httpcmd.RecursorErrCodeStart+4, "operate non-exist root zone")
)
