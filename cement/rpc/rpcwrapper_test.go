package rpc

import (
	"testing"
	"time"

	ut "cement/unittest"
)

type AddService struct {
	initNum int
}

func (service *AddService) Add(number int, result *int) error {
	*result = number + service.initNum
	return nil
}

func TestRPCWrapper(t *testing.T) {
	service := &AddService{
		initNum: 10,
	}

	endPoint := "127.0.0.1:5555"
	server, err := RunRpcServer(service, endPoint)
	ut.Assert(t, err == nil, "rpc server should lanuch successfully")
	defer server.Close()

	client, err := RpcClient(endPoint, 10*time.Second)
	ut.Assert(t, err == nil, "rpc client should create successfully")
	defer client.Close()

	var result int
	client.RpcCall("AddService.Add", 10, &result)
	ut.Equal(t, result, 20)
	client.RpcCall("AddService.Add", 20, &result)
	ut.Equal(t, result, 30)
}
