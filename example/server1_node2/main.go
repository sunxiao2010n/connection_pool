package main

import (
	"fmt"
	_ "net/http/pprof"
	"sort_service/cp-go/conn_pools"
	"sort_service/cp-go/server"

	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
)

func main() {
	ip, _ := conn_pools.LocalIP()
	host := fmt.Sprint(ip)
	port := "22345"
	ipAddr := fmt.Sprintf("%v:%v", host, port)
	logs.Info("local ip address is %v", ipAddr)

	// conn_pools.SetZkHosts("localhost:2181")
	// conn_pools.ConnPoolAdd("base_recall", "/service/recall_service/base_recall")
	// //conn_pools.ConnPoolAdd("service_name", "/test/go-client")
	// conn_pools.Start()
	conn_pools.Init("localhost:2181", map[string]string{"base_recall": "/service/recall_service/base_recall"})

	//conn_pools.Install("/services0000000000")
	go server.StartServer(host, port)

	var conn *zk.Conn

	// select {
	// case <-time.After(5 * time.Second):
	// 	//conn = conn_pools.TestCreate("/service/recall_service/base_recall", ipAddr)
	// 	// conn = conn_pools.TestCreate("/service/recall_service/base_recall", ipAddr)
	// 	conn.Close()
	// }

	//select {
	//case <-time.After(20 * time.Second):
	//	conn_pools.TestDelete("/services0000000000", ipAddr)
	//}

	//select {
	//case <-time.After(5 * time.Second):
	//	conn_pools.ClientRequestExample()
	//}

	_ = conn
	select {}
}
