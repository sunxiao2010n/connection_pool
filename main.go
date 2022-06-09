package main

import (
	"fmt"
	_ "net/http/pprof"
	"sort_service/cp-go/conn_pools"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
)

func main() {
	ip, _ := conn_pools.LocalIP()
	host := fmt.Sprint(ip)
	port := "32345"
	ipAddr := fmt.Sprintf("%v:%v", host, port)
	logs.Info("local ip address is %v", ipAddr)

	// conn_pools.SetZkHosts("localhost:2181")
	// conn_pools.ConnPoolAdd("base_recall-or-whatever-is-named-in-your-service", "/service/recall_service/base_recall")
	// conn_pools.Start()
	conn_pools.Init("localhost:2181", map[string]string{"base_recall-or-whatever-is-named-in-your-service": "/service/recall_service/base_recall"})

	//conn_pools.Install("/services0000000000")
	//go server.StartServer(host, port)

	var conn *zk.Conn
	_ = conn

	//serviceName := "base_recall"
	//ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
	//defer cancel()
	//for i := 0; i < 1; i++ {
	//	ctx1:=context.WithValue(ctx,conn_pools.REQUESTID,fmt.Sprint("r1-",i))
	//	go conn_pools.RemoteCall("base_recall-or-whatever-is-named-in-your-service", rpc.NewSayHelloClient, (*rpc.SayHelloClient).SayHello, ctx1, &rpc.HelloReq{})
	//}
	//
	//ctx2,cancel2:=context.WithTimeout(context.Background(),time.Second*10)
	//defer cancel2()
	//
	//for i := 0; i < 1; i++ {
	//	go conn_pools.RemoteCall("base_recall-or-whatever-is-named-in-your-service", rpc.NewSayHelloClient, (*rpc.SayHelloClient).SayHello, ctx2, &rpc.HelloReq{})
	//}

	time.Sleep(time.Second * 1)
	// conn_pools.ClientRequestExample()
	select {}
}
