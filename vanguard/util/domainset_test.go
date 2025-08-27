package util

import (
	ut "cement/unittest"
	"g53"

	"testing"
)

func TestDomainSet(t *testing.T) {
	s := NewDomainSet()

	n1, _ := g53.NewName("www.baidu.cn.", false)
	s.Add(n1)

	n2, _ := g53.NewName("www.baidU.cn.", false)
	ut.Assert(t, s.Include(n2), "baidu.cn is in set")

	n2, _ = g53.NewName("www.baidU.cn", false)
	ut.Assert(t, s.Include(n2), "baidu.cn is in set")

	n2, _ = g53.NewName("www.baidU.ocn", false)
	ut.Assert(t, s.Include(n2) == false, "baidu.ocn is not in set")
}
