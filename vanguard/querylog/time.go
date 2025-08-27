package querylog

import (
	"time"
)

const (
	secToNanoSec      = 1000000000
	miliSecToNanoSec  = 1000000
	microSecToNanoSec = 1000
)

var monthStrs []string = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func GetMonthString(month time.Month) string {
	return monthStrs[month-1]
}

func GetAllTimeArgs() (int, time.Month, int, int, int, int, int, int64) {
	now := time.Now()

	year, month, day := now.Date()
	hour, min, sec := now.Clock()
	tsNano := now.UnixNano()
	ms := int(tsNano % secToNanoSec / miliSecToNanoSec)

	return year, month, day, hour, min, sec, ms, tsNano
}
