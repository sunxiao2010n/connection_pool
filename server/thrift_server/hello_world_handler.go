package thrift_server

import (
	rpc "connection_pool/thrift_rpc_definitions/gen-go"
	"context"
	"github.com/astaxie/beego/logs"
)

type SayHelloHandler struct{}

func (h *SayHelloHandler) SayHello(ctx context.Context, req *rpc.HelloReq) (r *rpc.HelloRsp, err error) {
	logs.Info("Here I am server, received hello from client, and respond")
	rsp := rpc.NewHelloRsp()
	return rsp, nil
}
