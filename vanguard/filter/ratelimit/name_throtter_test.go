package ratelimit

import (
	"testing"
	"time"

	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/logger"
	view "vanguard/viewselector"
)

func TestNameFilter(t *testing.T) {
	logger.UseDefaultLogger("debug")
	var conf config.VanguardConf
	conf.Filter.DomainNameLimit = []config.DomainNameRateLimitInView{
		config.DomainNameRateLimitInView{
			View: "default",
		},
	}
	view.NewSelectorMgr(&conf)
	throtter := NewNameThrotter(&conf)
	_, err := throtter.addNameRateLimit("default", "*.com", 10)
	ut.Assert(t, err == nil, "name rrl add should succeed")
	_, err = throtter.addNameRateLimit("default", "c.com", 20)
	ut.Assert(t, err == nil, "name rrl add should succeed")
	_, err = throtter.addNameRateLimit("default", "cn.", 30)
	ut.Assert(t, err == nil, "name rrl add should succeed")
	name, _ := g53.NameFromString("b.c.com.")
	for i := 0; i < 10; i++ {
		ut.Assert(t, throtter.IsNameAllowed("default", name), "name isn't exceed the threshhold")
	}
	ut.Assert(t, !throtter.IsNameAllowed("default", name), "name exceed the threshhold")
	name, _ = g53.NameFromString("c.c.com.")
	ut.Assert(t, !throtter.IsNameAllowed("default", name), "name exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsNameAllowed("default", name), "one second has passed")

	name, _ = g53.NameFromString("c.com.")
	for i := 0; i < 20; i++ {
		ut.Assert(t, throtter.IsNameAllowed("default", name), "name isn't exceed the threshhold")
	}
	ut.Assert(t, !throtter.IsNameAllowed("default", name), "name exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsNameAllowed("default", name), "one second has passed")

	name, _ = g53.NameFromString("a.cn.")
	for i := 0; i < 1000; i++ {
		ut.Assert(t, throtter.IsNameAllowed("default", name), "name has no limits")
	}

	name, _ = g53.NameFromString("cn.")
	for i := 0; i < 30; i++ {
		ut.Assert(t, throtter.IsNameAllowed("default", name), "name has no limits")
	}
	ut.Assert(t, !throtter.IsNameAllowed("default", name), "name exceed the threshhold")
	<-time.After(time.Second)
	ut.Assert(t, throtter.IsNameAllowed("default", name), "one second has passed")
}
