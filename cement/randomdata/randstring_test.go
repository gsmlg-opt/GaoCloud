package randomdata

import (
	"sort"
	"testing"

	ut "cement/unittest"
	"cement/set"
)

func TestRandString(t *testing.T) {
	generatedStrs := set.NewSet()
	for i := 0; i < 100000; i++ {
		str := RandString(10)
		ut.Assert(t, generatedStrs.Contains(str) == false, "string is duplciated")
		generatedStrs.Add(str)
	}

	strCount := 10
	for i := 0; i < 100000; i++ {
		uniqueStrs, _ := UniqueRandStrings(5, strCount)
		ut.Equal(t, len(uniqueStrs), strCount)
		sort.Strings(uniqueStrs)
		for i := 1; i < strCount; i++ {
			ut.Assert(t, uniqueStrs[i] != uniqueStrs[i-1], "no string should equal")
		}
	}

	_, err := UniqueRandStrings(1, 27)
	ut.Assert(t, err != nil, "26 char can only generate 26 unique strings")
}
