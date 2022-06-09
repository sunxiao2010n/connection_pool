package conn_pools

import (
	"os"

	"github.com/astaxie/beego/logs"
)

func start() {
	turnOnGracefulExit()
}

func gracefulExit() {
	logs.Info("begin cleaning pools")
	disconnectAll()
	logs.Info("pools cleaned")
	os.Exit(0)
}

func disconnectAll() {
	pools.Range(func(k, v interface{}) bool {
		pool, ok := v.(*pool)
		if !ok {
			logs.Error("error: value of pool %v in pools is %v", k, v)
		}
		logs.Info("current node count : %v, pool %s ready to go offline, ", len(pool.nodes), pool.name)
		for _, node := range pool.nodes {
			connCount := 0
			loop := true
			for loop {
				select {
				case c := <-node.poolChan:
					if err := c.close(); err != nil {
						logs.Error(err)
					}
					connCount++
				default:
					loop = false
				}
			}
			logs.Info("%v connections to node %v closed", connCount, node.name)
		}
		return true
	})
}
