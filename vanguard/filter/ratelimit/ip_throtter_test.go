package ratelimit

import (
	"net"
	"testing"
	"time"

	ut "cement/unittest"
	"vanguard/config"
	"vanguard/logger"
)

func TestIPFilter(t *testing.T) {
	logger.UseDefaultLogger("debug")
	var conf config.VanguardConf
	throtter := NewIPThrotter(&conf)
	_, err := throtter.addIpRateLimit("192.168.0.0/16", 10)
	ut.Assert(t, err == nil, "add ip rrl should succeed")
	_, err = throtter.addIpRateLimit("192.168.3.0/24", 20)
	ut.Assert(t, err == nil, "add ip rrl should succeed")
	ip := net.ParseIP("192.168.3.1")
	for i := 0; i < 20; i++ {
		ut.Assert(t, throtter.IsIPAllowed(ip), "ip isn't exceed the threshhold")
	}
	ut.Assert(t, !throtter.IsIPAllowed(ip), "ip exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsIPAllowed(ip), "one second has passed")

	ip = net.ParseIP("192.168.4.1")
	for i := 0; i < 10; i++ {
		ut.Assert(t, throtter.IsIPAllowed(ip), "ip isn't exceed the threshhold")
	}
	ut.Assert(t, !throtter.IsIPAllowed(ip), "ip exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsIPAllowed(ip), "one second has passed")
	<-time.After(time.Second)

	_, err = throtter.updateIpRateLimit("192.168.0.0/16", 10)
	ut.Assert(t, err == nil, "update ip rrl should succeed")
	for i := 0; i < 10; i++ {
		ut.Assert(t, throtter.IsIPAllowed(ip), "ip isn't exceed the threshhold")
	}
	ut.Assert(t, !throtter.IsIPAllowed(ip), "ip exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsIPAllowed(ip), "one second has passed")
}

func BenchmarkIPRadixSearch(b *testing.B) {
	b.StopTimer()
	logger.UseDefaultLogger("debug")
	logger.UseDefaultLogger("debug")
	var conf config.VanguardConf
	throtter := NewIPThrotter(&conf)
	throtter.addIpRateLimit("192.168.0.0/16", 10)
	throtter.addIpRateLimit("192.168.3.0/24", 20)
	ip := net.ParseIP("192.169.3.1")
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		throtter.IsIPAllowed(ip)
	}
	b.StopTimer()
}
