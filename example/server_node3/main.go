package main

import (
	"connection_pool/conn_pools"
	"connection_pool/server"
	"connection_pool/zk_tool"
	_ "connection_pool/zk_tool"
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
	zt.Init("192.168.216.130")
	zt.CreateZPath(ipAddr, zk.FlagEphemeral)

	go server.StartServer(host, port)

	select {}
}
