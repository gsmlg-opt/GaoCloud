package querylog

import (
	"bufio"
	"g53"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	ut "cement/unittest"
	"vanguard/config"
	"vanguard/core"
)

const (
	lines = 1000000
)

func TestQuerylog(t *testing.T) {
	clientStr := "client"
	ipStr := "1.1.1.1"
	portStr := "1000"
	sep1 := ":"
	viewStr := "view"
	defaultStr := "default:"
	nameStr := "a.com"
	classStr := "IN"
	typeStr := "A"
	rcodeStr := "NOERROR"
	rcStr := "+"
	signedStr := "NS"
	ednsStr := "E"
	tcpStr := "NT"
	ndStr := "ND"
	ncStr := "NC"
	hitStr := "H"

	var conf config.VanguardConf

	os.Remove("./q.log.")
	conf.Logger.Querylog.Path = "./q.log."
	conf.Logger.Querylog.FileSize = 2000000000
	conf.Logger.Querylog.Versions = 1
	conf.Logger.Querylog.Extension = true

	ql := NewQuerylog(&conf).(*QueryLogger)

	name, err := g53.NameFromString("a.com")
	ut.Assert(t, err == nil, "failed to create name")

	addr, err := net.ResolveUDPAddr("udp", ipStr+":"+portStr)
	ut.Assert(t, err == nil, "failed to create addr")

	response := g53.MakeQuery(name, g53.RR_A, 512, false)
	ut.Assert(t, response != nil, "failed to create response")
	client := core.Client{Request: response,
		Response:   response,
		CreateTime: time.Now(),
		CacheHit:   true,
		View:       "default",
		Addr:       addr}

	for i := 0; i < lines; i++ {
		ql.LogWrite(client)
	}

	logFile, err := os.Open(conf.Logger.Querylog.Path)
	ut.Assert(t, err == nil, "failed to open log file")
	defer logFile.Close()

	fileReader := bufio.NewReader(logFile)
	ut.Assert(t, fileReader != nil, "failed to create reader")
	i := 0
	for {
		logStr, err := fileReader.ReadString('\n')
		if err != nil {
			break
		}
		strs := strings.Split(logStr, " ")
		ut.Assert(t, strs[2] == clientStr, "incorrect client string")
		ut.Assert(t, strs[3] == ipStr, "incorrect ip string")
		ut.Assert(t, strs[4] == portStr+sep1, "incorrect port string")
		ut.Assert(t, strs[5] == viewStr, "incorrect view string")
		ut.Assert(t, strs[6] == defaultStr, "incorrect default string")
		ut.Assert(t, strs[7] == nameStr, "incorrect name string")
		ut.Assert(t, strs[8] == classStr, "incorrect class string")
		ut.Assert(t, strs[9] == typeStr, "incorrect type string")
		ut.Assert(t, strs[10] == rcodeStr, "incorrect rcode string")
		ut.Assert(t, strs[11] == rcStr, "incorrect rc string")
		ut.Assert(t, strs[12] == signedStr, "incorrect signed string")
		ut.Assert(t, strs[13] == ednsStr, "incorrect edns string")
		ut.Assert(t, strs[14] == tcpStr, "incorrect tcp string")
		ut.Assert(t, strs[15] == ndStr, "incorrect nd string")
		ut.Assert(t, strs[16] == ncStr, "incorrect nc string")
		ut.Assert(t, strs[17] == hitStr, "incorrect hit string")
		i++
	}
	ut.Assert(t, i > lines-100 && i <= lines, "incorrect number of lines")
}

func BenchmarkQuerylog(b *testing.B) {
	b.StopTimer()

	var conf config.VanguardConf

	os.Remove("./q.log.")
	conf.Logger.Querylog.Path = "./q.log."
	conf.Logger.Querylog.FileSize = 50000000
	conf.Logger.Querylog.Versions = 5
	conf.Logger.Querylog.Extension = false

	ql := NewQuerylog(&conf).(*QueryLogger)
	name, err := g53.NameFromString("a.com")
	if err != nil {
		panic("failed to create name")
	}

	addr, err := net.ResolveUDPAddr("udp", "1.1.1.1:")
	if err != nil {
		panic("failed to create addr")
	}

	response := g53.MakeQuery(name, g53.RR_A, 512, false)
	client := core.Client{Request: response,
		CreateTime: time.Now(),
		CacheHit:   true,
		View:       "default",
		Addr:       addr}

	for i := 0; i < lines; i++ {
		ql.LogWrite(client)
	}

	b.StopTimer()
}
