package k8s

import (
	"testing"

	ut "cement/unittest"
)

func TestReverseName(t *testing.T) {
	cases := []struct {
		network       string
		hasError      bool
		reverseDomain string
	}{
		{"1.2.3.1/24", false, "3.2.1.in-addr.arpa"},
		{"1.2.3.1/16", false, "2.1.in-addr.arpa"},
		{"1.2.3.1/8", false, "1.in-addr.arpa"},
		{"1.2.3.1/22", true, ""},
		{"xx.2.3.1/22", true, ""},
		{"1111.2.3.1/24", true, ""},
	}

	for _, cas := range cases {
		q, err := reverseZoneNameForNetwork(cas.network)
		if cas.hasError {
			ut.Assert(t, err != nil, "%v should has error", cas)
		} else {
			ut.Equal(t, q, cas.reverseDomain)
		}
	}
}
