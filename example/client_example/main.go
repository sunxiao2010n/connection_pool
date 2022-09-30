package main

import (
	"connection_pool/conn_pools"
	rpc "connection_pool/thrift_rpc_definitions/gen-go"
	"context"
	"fmt"
	"github.com/astaxie/beego/logs"
	"time"
)

func main() {
	ip, _ := conn_pools.LocalIP()
	host := fmt.Sprint(ip)
	port := "12345"
	ipAddr := fmt.Sprintf("%v:%v", host, port)
	logs.Info("local ip address is %v", ipAddr)

	serviceName := "connection_pool_test-server"

	/*************
		change the following zkAddrs to the actual zk address:

		conn_pools.Init("192.168.216.130", map[string]string{serviceName: "/connection_pool_test/server"})
	**************/
	conn_pools.Init("192.168.216.130", map[string]string{serviceName: "/connection_pool_test/server"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	time.Sleep(time.Second * 1)

	r, err := conn_pools.RemoteCall(ctx, serviceName, rpc.NewHelloworldClient, (*rpc.HelloworldClient).SayHello, &rpc.HelloReq{})
	if err != nil {
		logs.Error(err)
	} else {
		logs.Info("remote call ok: %v\n", r)
	}

	logs.Info("press ctrl+c to exit.......\n")
	select {}
}
