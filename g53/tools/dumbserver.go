package main

import (
	"flag"
	"fmt"
	"g53"
	"g53/util"
	"net"
)

var (
	port int
)

func init() {
	flag.IntVar(&port, "p", 53, "dns server port default to 53")
}

func main() {
	flag.Parse()
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(fmt.Sprintf("bind port %d failed %s\n", port, err.Error()))
	}
	defer conn.Close()

	buf := make([]byte, 512)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		buffer := util.NewInputBuffer(buf[0:n])

		msg, err := g53.MessageFromWire(buffer)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			fmt.Printf("%s\n", msg.String())
			fmt.Printf("%s\n", util.BytesToElixirStr(buf[0:n]))
			conn.WriteTo(buf[0:n], addr)
		}
	}
}
