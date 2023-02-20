# connection_pool
  
这个连接池经过了两个游戏平台的打磨。目前在DAU超过10M的环境里表现稳定。  

Connection pool with service discovery.  
Choose Thrift or gRPC as the protocol. Custom protocol will be OK too.  
Default load balance method is weightedRoundRobin.  
  
  
Instructions：  
  
1.Run zookeeper  
  create a path on zk：  
  > '> zkCli.sh  
  > '> create /connection_pool_test  
  > '> create /connection_pool_test/server  
  
2.Start Example Server  
  > '> cd connection_pool/example/thrift_server_example  
  > edit main.go and change zookeeper address to the actual zk  
  > '> go run main.go  
  
3.Start Example Client  
  > '> cd connection_pool/example/client_example  
  > edit main.go and change zookeeper address to the actual zk  
  > '> go run main.go  

