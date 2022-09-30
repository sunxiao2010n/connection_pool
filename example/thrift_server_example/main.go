package main

import (
	"connection_pool/conn_pools"
	"connection_pool/server/thrift_server"
	"connection_pool/server/zk_tool"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
)

const (
	zkPath = "/connection_pool_test/server"
)

func main() {
	ip, _ := conn_pools.LocalIP()
	host := fmt.Sprint(ip)
	port := "32345"
	ipAddr := fmt.Sprintf("%v/%v:%v", zkPath, host, port)
	logs.Info("local ip address is %v", ipAddr)

	zt := &zk_tool.ZkTool{}

	/*************
		change the following zkAddrs to the actual zk address:

		zt.Init("192.168.216.130")
	**************/
	zt.Init("192.168.216.130")

	zt.CreateZPath(ipAddr, zk.FlagEphemeral)

	go thrift_server.StartServer(host, port)

	select {}
}
