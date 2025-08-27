package failforwarder

import (
	"vanguard/httpcmd"
)

var (
	ErrInvalidFailForwarder   = httpcmd.NewError(httpcmd.FailForwarderErrCodeStart, "fail forward addr isn't valid")
	ErrDuplicateFailForwarder = httpcmd.NewError(httpcmd.FailForwarderErrCodeStart+1, "duplicate fail forwarder")
	ErrNotExistFailForwarder  = httpcmd.NewError(httpcmd.FailForwarderErrCodeStart+2, "unknown fail forwarder")
)
