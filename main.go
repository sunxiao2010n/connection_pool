package main

import (
	"connection_pool/conn_pools"
	rpc "connection_pool/thrift_rpc_definitions/gen-go"
	"context"
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/astaxie/beego/logs"
)

func main() {
	ip, _ := conn_pools.LocalIP()
	host := fmt.Sprint(ip)
	port := "12345"
	ipAddr := fmt.Sprintf("%v:%v", host, port)
	logs.Info("local ip address is %v", ipAddr)

	// conn_pools.ConnPoolAdd("base_recall-or-whatever-is-named-in-your-service", "/service/recall_service/base_recall")
	// conn_pools.Start()
	serviceName := "connection_pool_test-server"
	conn_pools.Init("192.168.216.130:2181", map[string]string{serviceName: "/connection_pool_test/server"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	//for i := 0; i < 1; i++ {
	//	ctx1 := context.WithValue(ctx, "requestID", fmt.Sprint("r1-", i))
	//	ctx2 := context.WithValue(ctx1, "timeout", time.Second*5)

	time.Sleep(time.Second * 3)

	r, err := conn_pools.RemoteCall(ctx, serviceName, rpc.NewHelloworldClient, (*rpc.HelloworldClient).SayHello, &rpc.HelloReq{})
	if err != nil {
		logs.Error(err)
	} else {
		logs.Info("remote call ok: %v\n", r)
	}

	//
	//ctx2,cancel2:=context.WithTimeout(context.Background(),time.Second*10)
	//defer cancel2()
	//
	//for i := 0; i < 1; i++ {
	//	go conn_pools.RemoteCall(ctx, serviceName, rpc.NewSayHelloClient, (*rpc.SayHelloClient).SayHello, &rpc.HelloReq{})
	//}

	logs.Info("press ctrl+c to exit.......\n")
	select {}
}
