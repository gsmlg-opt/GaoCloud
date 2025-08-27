package localdata

import (
	"testing"

	ut "cement/unittest"
	"g53"
	"vanguard/core"
)

func buildClient(view, name string, typ g53.RRType) *core.Client {
	dname, _ := g53.NameFromString(name)
	query := g53.MakeQuery(dname, typ, 512, false)
	query.Header.Id = 1
	return &core.Client{
		View:    view,
		Request: query,
	}
}

func TestMessageCache(t *testing.T) {
	view := "v1"
	nxDomain := []string{"*.playboy.xxx.", "*.playboy.cn."}
	nxRRset := []string{"*.playboy.com.", "*.playboy.net."}
	exception := []string{"zdns.playboy.xxx.", "www.playboy.com."}
	redirect := []string{
		"cc.playboy.xxx. 60 A 1.1.1.1",
		"cc.playboy.xxx. 60 A 2.2.2.2",
		"cc.playboy.xxx. 60 A 3.3.3.3",
		"cc.playboy.com. 3600 AAAA ::2",
		"cc.playboy.com. 3600 AAAA ::3",
		"*.dd.playboy.xxx. 3600 AAAA ::1",
		"dd.playboy.cn. 3600 MX 5 mxbiz1.qq.com.",
	}

	localdata := NewLocalData()
	localdata.AddPolicies(view, LPNXDomain, nxDomain)
	localdata.AddPolicies(view, LPNXRRset, nxRRset)
	localdata.AddPolicies(view, LPExceptionDomain, exception)
	localdata.AddPolicies(view, LPLocalRRset, redirect)
	cli := buildClient("v1", "cccxx.playboy.xxx.", g53.RR_A)
	found := localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found, "playboy is in localdata")
	ut.Equal(t, cli.Response.Header.ANCount, uint16(0))
	ut.Equal(t, uint8(cli.Response.Header.Rcode), uint8(g53.R_NXDOMAIN))

	cli = buildClient("v1", "Zdns.playboy.xxx.", g53.RR_A)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found == false, "zdns.playboy is exception")

	cli = buildClient("v1", "cc.playboy.xxx.", g53.RR_A)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found, "cc.playboy has localdata")
	ut.Equal(t, cli.Response.Header.ANCount, uint16(3))
	ut.Equal(t, uint8(cli.Response.Header.Rcode), uint8(g53.R_NOERROR))

	cli = buildClient("v1", "a.dd.playboy.xxx.", g53.RR_AAAA)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found, "dd.playboy has localdata")
	ut.Equal(t, cli.Response.Header.ANCount, uint16(1))
	ut.Equal(t, uint8(cli.Response.Header.Rcode), uint8(g53.R_NOERROR))

	cli = buildClient("v1", "a.dd.playboy.xxx.", g53.RR_A)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found, "playboy has localdata")
	ut.Equal(t, cli.Response.Header.ANCount, uint16(0))
	ut.Equal(t, uint8(cli.Response.Header.Rcode), uint8(g53.R_NXDOMAIN))

	cli = buildClient("v2", "a.dd.playboy.xxx.", g53.RR_A)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found == false, "no view v2")
	ut.Assert(t, cli.Response == nil, "no response found")

	cli = buildClient("v1", "dd.playboy.cn.", g53.RR_MX)
	found = localdata.ResponseWithLocalData(cli)
	ut.Assert(t, found, "dd.playboy.cn has mx record")
	ut.Equal(t, cli.Response.Header.ANCount, uint16(1))
	ut.Equal(t, cli.Response.Sections[g53.AnswerSection][0].Rdatas[0].String(), "5 mxbiz1.qq.com.")

	localdata.RemovePolicies("v1", LPLocalRRset, []string{"cc.playboy.xxx. 60 A 2.2.2.2"})
	cli = buildClient("v1", "cc.playboy.xxx.", g53.RR_A)
	localdata.ResponseWithLocalData(cli)
	ut.Equal(t, cli.Response.Header.ANCount, uint16(2))

	localdata.RemovePolicies("v1", LPLocalRRset, []string{"cc.playboy.xxx. 60 A 1.1.1.1", "cc.playboy.xxx. 60 A 3.3.3.3"})
	cli = buildClient("v1", "cc.playboy.xxx.", g53.RR_A)
	localdata.ResponseWithLocalData(cli)
	ut.Equal(t, cli.Response.Header.ANCount, uint16(0))
	ut.Equal(t, uint8(cli.Response.Header.Rcode), uint8(g53.R_NXDOMAIN))
}
