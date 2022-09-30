# connection_pool
  
Connection pool with service discovery.  
Using protocol thrift or gRPC, and you can imply your own protocol.  
Default load balance method is weightedRoundRobin.  
  
  
这个连接池形成于某游戏平台。其性能和稳定性由@juxu007进行改进。目前运行在DAU在10M以上的环境。  
  
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

