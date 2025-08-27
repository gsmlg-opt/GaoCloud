package rpc

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"time"
)

func RunRpcServer(service interface{}, endPoint string) (io.Closer, error) {
	server := rpc.NewServer()

	if err := server.Register(service); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", endPoint)
	if err != nil {
		return nil, err
	}

	go server.Accept(listener)
	return listener, nil
}

type Client struct {
	*rpc.Client
	timeout time.Duration
}

func RpcClient(endPoint string, timeout time.Duration) (*Client, error) {
	errChan := make(chan error, 1)
	var client *rpc.Client
	var err error
	go func() {
		client, err = rpc.Dial("tcp", endPoint)
		errChan <- err
	}()

	select {
	case <-errChan:
	case <-time.After(timeout):
		return nil, fmt.Errorf("connect to rpc server timeout")
	}

	if err == nil {
		return &Client{
			Client:  client,
			timeout: timeout,
		}, nil
	} else {
		return nil, err
	}
}

func (client *Client) RpcCall(method string, arg interface{}, ret interface{}) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Call(method, arg, ret)
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(client.timeout):
		return fmt.Errorf("rpc call timeout")
	}
}
