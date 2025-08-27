package randomdata

import (
	"fmt"
	"net"
	"testing"

	ut "cement/unittest"
)

func TestRandIP(t *testing.T) {
	SeedUsingNow()
	for i := 0; i < 10000; i++ {
		ipStr := RandV4IP().String()
		ut.Assert(t, net.ParseIP(ipStr).To4() != nil, fmt.Sprintf("ip %s isn't valid", ipStr))
	}
}
