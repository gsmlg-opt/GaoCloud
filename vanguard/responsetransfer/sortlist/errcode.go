package sortlist

import (
	"vanguard/httpcmd"
)

var (
	ErrAddSortListFailed    = httpcmd.NewError(httpcmd.SortListErrCodeStart, "add sort list failed")
	ErrDeleteSortListFailed = httpcmd.NewError(httpcmd.SortListErrCodeStart+1, "delete sort list failed")
	ErrUpdateSortListFailed = httpcmd.NewError(httpcmd.SortListErrCodeStart+2, "update sort list failed")
)
