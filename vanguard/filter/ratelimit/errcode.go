package ratelimit

import (
	"vanguard/httpcmd"
)

var (
	ErrRrlExists                 = httpcmd.NewError(httpcmd.RateLimitErrCodeStart, "rrl policy already exist")
	ErrNonExistRrl               = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+1, "non-exist rrl policy")
	ErrTooMuchRrls               = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+2, "too much rrls(over 999)")
	ErrAddIPRateLimitFailed      = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+3, "add ip rate limit failed")
	ErrDeleteIPRateLimitFailed   = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+4, "delete ip rate limit failed")
	ErrUpdateIPRateLimitFailed   = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+5, "update ip rate limit failed")
	ErrAddNameRateLimitFailed    = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+6, "add name rate limit failed")
	ErrUpdateNameRateLimitFailed = httpcmd.NewError(httpcmd.RateLimitErrCodeStart+7, "update name rate limit failed")
)
