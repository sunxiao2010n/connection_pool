package server

import (
	rpc "connection_pool/thrift_rpc_definitions/gen-go"
	"context"
	"fmt"
	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
	"github.com/prometheus/common/log"
	"net"
	"os"

	"time"
)

func StartServer(host, port string) {
	logs.Info("start server at %v:%v", host, port)
	transport, err := thrift.NewTServerSocket(net.JoinHostPort(host, port))
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
	processor := rpc.NewHelloworldProcessor(&sh)

	server4 := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
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

	sayHello := rpc.NewHelloworldClient(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logs.Info("Send a test message from server itself")

	r, e := sayHello.SayHello(ctx, &rpc.HelloReq{})
	if e != nil {
		logs.Error(e)
	} else {
		logs.Info("OK! I received the response, content is %v", r)
	}
}
