package dns64

import (
	"testing"

	ut "cement/unittest"
	"net"
)

func TestConvertV4ToV6(t *testing.T) {
	cases := []struct {
		config     string
		v4         string
		expectedV6 string
	}{
		{"64:ff9b::/96 ", "192.0.2.1", "64:ff9b::192.0.2.1"},
		{"64:ff9b::/40 ::", "117.34.15.57", "64:ff9b:75:220f:39::"},
		{"64:ff9b::/64 ::", "117.34.15.57", "64:ff9b::75:220f:3900:0"},
		{"64:ff9b::/32 ::", "117.34.15.57", "64:ff9b:7522:f39::"},
		{"64:ff9b::/32 ::1", "117.34.15.57", "64:ff9b:7522:f39::1"},
	}

	for _, tc := range cases {
		converter, _ := converterFromString(tc.config)
		v4 := net.ParseIP(tc.v4).To4()
		expectedV6 := net.ParseIP(tc.expectedV6)
		v6 := converter.synthesisAddr(v4)
		ut.Equal(t, v6, expectedV6)
	}
}
