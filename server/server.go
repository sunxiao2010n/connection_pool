package server

import (
	"context"
	"fmt"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
	"github.com/prometheus/common/log"
	"net"
	"os"
	"sort_service/cp-go/thrift_rpc_definitions/gen-go/rpc"
	"time"
)

type SayHelloHandler struct{}

func (h *SayHelloHandler) SayHello(ctx context.Context, req *rpc.HelloReq) (r *rpc.HelloRsp, err error) {
	logs.Info("received hello")
	rsp := rpc.NewHelloRsp()
	return rsp, nil
}

func StartServer(host, port string) {
	transport, err := thrift.NewTServerSocket(net.JoinHostPort("0.0.0.0", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving address:", err)
		os.Exit(1)
	}

	//protocol
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	//no buffered
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())

	if err := transport.Open(); err != nil {
		fmt.Fprintln(os.Stderr, "Error opening socket", " ", err)
		os.Exit(1)
	}
	defer transport.Close()

	var sh SayHelloHandler

	server4 := thrift.NewTSimpleServer4(rpc.NewSayHelloProcessor(&sh), transport, transportFactory, protocolFactory)
	go server4.Serve()
	select {
	case <-time.After(time.Second):
		TestClientRequest(host, port)
	}
	select {}
}

func TestClientRequest(ip, port string) {
	socket, err := thrift.NewTSocket(net.JoinHostPort(ip, port))
	if err != nil {
		log.Error("error resolving address:", err)
		os.Exit(1)
	}

	//protocol
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	//no buffered
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())

	transport, err := transportFactory.GetTransport(socket)
	if err != nil {
		log.Error("error running client:", err)
	}

	if err := transport.Open(); err != nil {
		log.Error("error running client:", err)
	}

	iProto := protocolFactory.GetProtocol(transport)
	oProto := protocolFactory.GetProtocol(transport)

	client := thrift.NewTStandardClient(iProto, oProto)

	sayHello := rpc.NewSayHelloClient(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	r, e := sayHello.SayHello(ctx, &rpc.HelloReq{})
	logs.Info("err:%v,rsp:%v", e, r)
}
