package conn_pools

import (
	"sort"
	"strings"

	"github.com/astaxie/beego/logs"
)

const ()

type node struct {
	name       string
	value      string
	weightCtrl weightControl
	fuseCtrl   fuseControl
	poolChan   chan *thriftClient
}

func syncChildren(serviceName, path string, children, values []string) {
	sort.Strings(children)
	newNode := func(nodeName string) *node {
		n := new(node)
		n.name = nodeName
		n.value = nodeName
		n.poolChan = make(chan *thriftClient, poolSize)
		n.weightCtrl.weight = 1
		n.weightCtrl.effectiveWeight = n.weightCtrl.weight
		n.weightCtrl.quitChan = make(chan struct{})
		n.fuseCtrl.nodeName = nodeName
		n.fuseCtrl.poolName = serviceName
		go n.weightCtrl.nodeWeightIncrease(nodeName)
		return n
	}

	closeNode := func(n *node) {
		close(n.weightCtrl.quitChan)
		loop := true
		for loop {
			select {
			case c := <-n.poolChan:
				if err := c.close(); err != nil {
					logs.Error(err)
				}
			default:
				loop = false
			}
		}
		close(n.poolChan)
	}

	var tempNodes []*node
	tempPool := &pool{
		name:     serviceName,
		value:    path,
		fuseCtrl: fuseControl{poolName: serviceName},
		quitChan: make(chan struct{}),
	}
	p, found := pools.Load(serviceName)
	if found {
		pool, ok := p.(*pool)
		if !ok {
			logs.Error("error: value of pool %s in pools is %v", serviceName, p)
		}
		i, j := 0, 0
		for i < len(children) && j < len(pool.nodes) {
			cmp := strings.Compare(children[i], pool.nodes[j].name)
			if cmp == 0 { // renew
				tempNodes = append(tempNodes, pool.nodes[j])
				i, j = i+1, j+1
			} else if cmp < 0 { // add
				tempNodes = append(tempNodes, newNode(children[i]))
				i += 1
			} else { // delete
				closeNode(pool.nodes[j])
				j += 1
			}
		}
		for ; i < len(children); i++ {
			tempNodes = append(tempNodes, newNode(children[i]))
		}
		for ; j < len(pool.nodes); j++ {
			closeNode(pool.nodes[j])
		}
	} else {
		for i := 0; i < len(children); i++ {
			tempNodes = append(tempNodes, newNode(children[i]))
		}
	}

	tempPool.Lock()
	tempPool.nodes = tempNodes
	tempPool.Unlock()

	pools.Store(serviceName, tempPool)
}

func putConn(n *node, c *thriftClient) {
	select {
	case <-n.weightCtrl.quitChan:
		if err := c.close(); err != nil {
			logs.Error(err)
		}
		logs.Info("error: node %s is offline", n.name)
	default:
		if len(n.poolChan) >= int(poolSize) {
			logs.Error("error: pool %s is full", n.name)
			if err := c.close(); err != nil {
				logs.Error(err)
			}
		} else {
			n.poolChan <- c
		}
	}
}
