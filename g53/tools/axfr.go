package main

import (
	"flag"
	"fmt"
	"g53"
	"g53/util"
	"strings"
)

var (
	server string
	zone   string
	tsig   string
)

func init() {
	flag.StringVar(&server, "i", "", "dns server to query")
	flag.StringVar(&zone, "z", "", "zone name")
	flag.StringVar(&tsig, "y", "", "tsig key at name:secret format")
}

func main() {
	flag.Parse()

	conn, err := util.NewTCPConn(server)
	if err != nil {
		panic("connect to server failed:" + err.Error())
	}
	defer conn.Close()

	zoneName, err := g53.NewName(zone, true)
	if err != nil {
		panic("invalid zone name to query:" + err.Error())
	}

	nameAndSec := strings.Split(tsig, ":")
	if len(nameAndSec) != 2 {
		panic("tsig key isn't at name:secret format")
	}

	tsig, err := g53.NewTSIG(nameAndSec[0], nameAndSec[1], "hmac-md5")
	if err != nil {
		panic("invalid tsig key:" + err.Error())
	}

	msg := g53.MakeAXFR(zoneName, tsig)
	fmt.Printf("send query: %s\n", msg.String())
	render := g53.NewMsgRender()
	msg.Rend(render)

	if err := util.TCPWrite(render.Data(), conn); err != nil {
		panic("send query failed:" + err.Error())
	}

	answerBuffer, err := util.TCPRead(conn)
	if err != nil {
		panic("read axfr result failed:" + err.Error())
	}

	answer, err := g53.MessageFromWire(util.NewInputBuffer(answerBuffer))
	if err == nil {
		fmt.Printf(answer.String())
	} else {
		fmt.Printf("get err %s\n", err.Error())
	}
}
