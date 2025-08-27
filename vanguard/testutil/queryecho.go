package testutil

import (
	"net"

	"g53"
	"g53/util"
)

type Query struct {
	raw  *util.InputBuffer
	addr *net.UDPAddr
}

type Server struct {
	conn      *net.UDPConn
	queryChan chan Query
	stopChan  chan struct{}
}

func NewServer(addr string) (*Server, error) {
	addr_, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr_)
	if err != nil {
		return nil, err
	}

	return &Server{
		conn:      conn,
		stopChan:  make(chan struct{}),
		queryChan: make(chan Query),
	}, nil
}

func (s *Server) Run() {
	s.startHandlerRoutine()
	for {
		buf := make([]byte, 512)
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err == nil {
			buffer := util.NewInputBuffer(buf[0:n])
			s.queryChan <- Query{buffer, addr}
		}
	}
}

func (s *Server) startHandlerRoutine() {
	for i := 0; i < 50; i++ {
		go s.echo()
	}
}

func (s *Server) echo() {
	render := g53.NewMsgRender()
	for {
		select {
		case <-s.stopChan:
			s.stopChan <- struct{}{}
			return
		case query := <-s.queryChan:
			msg, err := g53.MessageFromWire(query.raw)
			if err != nil {
				continue
			}

			resp := msg.MakeResponse()
			ra1, _ := g53.AFromString("1.1.1.1")
			resp.AddRRset(g53.AnswerSection,
				&g53.RRset{
					Name:   msg.Question.Name,
					Type:   g53.RR_A,
					Class:  g53.CLASS_IN,
					Ttl:    g53.RRTTL(3600),
					Rdatas: []g53.Rdata{ra1},
				})

			resp.Rend(render)
			s.conn.WriteTo(render.Data(), query.addr)
			render.Clear()
		}
	}
}

func (s *Server) Stop() {
	s.conn.Close()
	s.stopChan <- struct{}{}
	<-s.stopChan
}
