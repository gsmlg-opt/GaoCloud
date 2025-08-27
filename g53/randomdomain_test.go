package g53

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	ut "cement/unittest"
)

func TestRandomName(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	maxDuplicateCount := 2

	names := make(map[string]int)
	duplicateCount := 0
	for i := 0; i < 1000; i++ {
		n := RandomNoneFQDNDomain()
		ut.Assert(t, strings.HasSuffix(n, ".") == false, "")
		_, err := NameFromString(n)
		ut.Assert(t, err == nil, "get err %v", err)
		_, ok := names[n]
		if ok {
			duplicateCount += 1
		}
		names[n] = i
	}
	ut.Assert(t, duplicateCount < maxDuplicateCount, "")
}
