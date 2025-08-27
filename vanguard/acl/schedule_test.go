package acl

import (
	"time"

	ut "cement/unittest"
	"testing"
)

func timeFromString(timeString string) time.Time {
	const longForm = "Jan 2, 2006 at 3:04pm"
	t, _ := time.Parse(longForm, timeString)
	return t
}

func TestTimeRange(t *testing.T) {
	cases := []struct {
		rangeStart    string
		rangeEnd      string
		timeToCheck   time.Time
		shouldInclude bool
	}{
		{"11:30", "14:20", timeFromString("Jan 2, 2017 at 11:40am"), true},
		{"11:30", "14:20", timeFromString("Jan 2, 2017 at 10:40am"), false},
		{"11:30", "14:20", timeFromString("Jan 2, 2017 at 14:20am"), false},
		{"15:30", "02:20", timeFromString("Jan 2, 2017 at 01:40am"), true},
		{"15:30", "02:20", timeFromString("Jan 2, 2017 at 02:19am"), true},
		{"15:30", "02:20", timeFromString("Jan 2, 2017 at 02:21am"), false},
		{"15:30", "02:20", timeFromString("Jan 2, 2017 at 11:21am"), false},

		//Oct 9, 2017 == Monday
		{"1 11:30", "2 2:20", timeFromString("Oct 9, 2017 at 11:40am"), true},
		{"1 11:30", "2 2:20", timeFromString("Oct 9, 2017 at 11:20am"), false},
		{"1 11:30", "2 2:20", timeFromString("Oct 10, 2017 at 1:40am"), true},
		{"1 11:30", "2 2:20", timeFromString("Oct 8, 2017 at 11:40am"), false},
		{"0 11:30", "0 12:20", timeFromString("Oct 8, 2017 at 11:40am"), true},
		{"4 11:30", "0 12:20", timeFromString("Oct 8, 2017 at 11:40am"), true},
		{"4 11:30", "0 12:20", timeFromString("Oct 8, 2017 at 13:40pm"), false},
		{"4 11:30", "0 12:20", timeFromString("Oct 13, 2017 at 11:31am"), true},

		{"10 9 11:30", "10 10 11:30", timeFromString("Oct 9, 2017 at 11:40am"), true},
		{"10 9 11:30", "10 10 11:30", timeFromString("Oct 9, 2017 at 11:20am"), false},
		{"10 9 11:30", "10 10 11:30", timeFromString("Oct 10, 2017 at 11:20am"), true},
		{"10 9 11:30", "10 10 11:30", timeFromString("Oct 11, 2017 at 11:20am"), false},
		{"10 9 11:30", "1 10 11:30", timeFromString("Oct 11, 2017 at 11:20am"), true},
		{"10 9 11:30", "1 10 11:30", timeFromString("Jan 9, 2017 at 11:29am"), true},
	}

	for _, tc := range cases {
		timeRange, err := rangeBuilder(tc.rangeStart, tc.rangeEnd)
		ut.Assert(t, err == nil, "time range should valid but get %v", err)
		ut.Equal(t, tc.shouldInclude, timeRange.IncludeTime(tc.timeToCheck))
	}
}
