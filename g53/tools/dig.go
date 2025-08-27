package main

import (
	"flag"
	"fmt"
	"g53"
	"g53/util"
	"net"
)

var (
	port   int
	typ    string
	subnet string
)

func init() {
	flag.IntVar(&port, "p", 53, "dns server port default to 53")
	flag.StringVar(&typ, "t", "a", "query type")
	flag.StringVar(&subnet, "s", "", "subnet")
}

func main() {
	flag.Parse()
	name := flag.Arg(1)
	addr := fmt.Sprintf("%s:%d", flag.Arg(0), port)

	fmt.Printf(">> dig %s %s %s\n", addr, name, typ)
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic("resolver failed:" + addr)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic("connect to server failed")
	}
	defer conn.Close()

	qn, err := g53.NewName(name, false)
	if err != nil {
		panic("invalid name to query:" + err.Error())
	}
	qtype, err := g53.TypeFromString(typ)
	if err != nil {
		panic("invalid type to query:" + err.Error())
	}
	msg := g53.MakeQuery(qn, qtype, 4096, false)
	msg.Header.Id = 1200
	if subnet != "" {
		msg.Edns.AddSubnetV4(subnet)
	}

	render := g53.NewMsgRender()
	msg.Rend(render)
	conn.Write(render.Data())
	fmt.Printf("%s\n", util.BytesToElixirStr(render.Data()))
	fmt.Printf(msg.String())

	answerBuffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(answerBuffer)
	if err == nil && n > 0 {
		answer, err := g53.MessageFromWire(util.NewInputBuffer(answerBuffer))
		if err == nil {
			fmt.Printf(answer.String())
		} else {
			fmt.Printf("get err %s\n", err.Error())
		}
	} else {
		fmt.Printf("get err %s\n", err.Error())
	}
}
