package forwarder

import (
	"vanguard/httpcmd"
)

var (
	ErrUnknownForwardStyle     = httpcmd.NewError(httpcmd.ForwarderErrCodeStart, "unknown forward style")
	ErrForwarderExists         = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+1, "forwarder exists")
	ErrNonExistForwarder       = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+2, "operate non-exist forwarder")
	ErrForwardZoneExists       = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+3, "forward zone with same name already exists")
	ErrDuplicateForwardZone    = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+4, "duplicate forward zone")
	ErrUnknownForcedForward    = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+5, "unknown forced forward option")
	ErrNoNeedForceForward      = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+6, "no need force forward option")
	ErrForwardZoneNotExists    = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+7, "forward zone not exist")
	ErrAddForwardZoneFailed    = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+8, "add forward zone failed")
	ErrDeleteForwardZoneFailed = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+9, "delete forward zone failed")
	ErrUpdateForwardZoneFailed = httpcmd.NewError(httpcmd.ForwarderErrCodeStart+10, "update forward zone failed")
)
