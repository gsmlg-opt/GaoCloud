package main

import (
	"flag"
	"fmt"
	"g53"
	"g53/util"
	"net"
	"time"
)

var (
	port int
)

func init() {
	flag.IntVar(&port, "p", 53, "dns server port default to 53")
}

type Client struct {
	req    []byte
	addr   *net.UDPAddr
	answer []byte
}

type Forwarder struct {
	*util.TimeoutConn
	Addr string
}

func NewForwarder(addr string, timeout time.Duration) (*Forwarder, error) {
	conn, err := util.NewClientTimeoutConn(addr, timeout)
	if err != nil {
		return nil, err
	}

	return &Forwarder{
		TimeoutConn: conn,
		Addr:        addr,
	}, nil
}

func forwardClient(s *Forwarder, clientChan <-chan Client, answerChan chan<- Client) {
	for {
		client := <-clientChan
		s.Write(client.req)

		answerBuffer := make([]byte, 512)
		n, _, err := s.ReadFromUDP(answerBuffer)
		if err == nil && n > 0 {
			answer, err := g53.MessageFromWire(util.NewInputBuffer(answerBuffer[0:n]))
			if err == nil {
				fmt.Printf("get answer %s\n", answer.String())
				client.answer = answerBuffer[0:n]
				answerChan <- client
			} else {
				fmt.Printf("reply get error %s\n", err.Error())
			}
		}
	}
}

func main() {
	flag.Parse()
	conn, err := util.NewServerTimeConn(fmt.Sprintf(":%d", port), time.Second)
	if err != nil {
		panic(fmt.Sprintf("bind port %d failed %s\n", port, err.Error()))
	}
	defer conn.Close()

	forwarder, err := NewForwarder("114.114.114.114:53", time.Second)
	if err != nil {
		panic(fmt.Sprintf("create forwarder failed %s\n", err.Error()))
	}

	clientChan := make(chan Client)
	answerChan := make(chan Client)
	go replayAnswers(conn, answerChan)
	go forwardClient(forwarder, clientChan, answerChan)
	for {
		buf := make([]byte, 512)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		buffer := util.NewInputBuffer(buf[0:n])
		msg, err := g53.MessageFromWire(buffer)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			fmt.Printf("get request: %s\n", msg.Question.String())
			clientChan <- Client{
				req:  buf[0:n],
				addr: addr,
			}
		}
	}
}

func replayAnswers(conn *util.TimeoutConn, answerChan <-chan Client) {
	for {
		client := <-answerChan
		conn.WriteTo(client.answer, client.addr)
	}
}
