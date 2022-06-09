package conn_pools

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego/logs"
)

// weight control configuration
const (
	maxWeight      = int32(100)
	intervalWeight = int32(5)
)

type weightControl struct {
	sync.RWMutex
	weight          int32
	effectiveWeight int32
	currentWeight   int32
	quitChan        chan struct{}
}

func weightedRoundRobin(pool *pool) (*node, error) {
	pool.Lock()
	defer pool.Unlock()
	totalWeight := int32(0)
	currentWeight := int32(0)
	selectedIndex := -1

	updateWeight := func(i int) {
		node := pool.nodes[i]
		totalWeight += node.weightCtrl.effectiveWeight
		node.weightCtrl.currentWeight += node.weightCtrl.effectiveWeight
		if node.weightCtrl.currentWeight >= currentWeight {
			currentWeight = node.weightCtrl.currentWeight
			selectedIndex = i
		}
	}

	for i := range pool.nodes {
		updateWeight(i)
	}

	if selectedIndex >= 0 {
		pool.nodes[selectedIndex].weightCtrl.currentWeight -= totalWeight
		return pool.nodes[selectedIndex], nil
	}

	return nil, fmt.Errorf("no node of pool %s selected", pool.name)
}

func (w *weightControl) onSuccess() {
	if atomic.LoadInt32(&w.effectiveWeight) < w.weight {
		atomic.AddInt32(&w.effectiveWeight, 3)
	}
}

func (w *weightControl) onReject() {
	if atomic.LoadInt32(&w.effectiveWeight) >= 3 {
		atomic.AddInt32(&w.effectiveWeight, -3)
	}
}

func (w *weightControl) nodeWeightIncrease(nodeName string) {
	// logs.Info("node %s weight begins to change", nodeName)
	interval := 5 * time.Second
	for {
		select {
		case <-time.After(interval):
			w.Lock()
			interval = 1500 * time.Millisecond
			// weightBefore := w.weight
			w.weight += intervalWeight
			w.effectiveWeight += intervalWeight
			if w.weight >= maxWeight {
				w.weight = maxWeight
				w.Unlock()
				logs.Info("node %s weight changed: %v -> %v", nodeName, 0, w.weight)
				return
			}
			// logs.Info("node %s weight changed: %v->%v", nodeName, weightBefore, w.weight)
			w.Unlock()
		case <-w.quitChan:
			logs.Info("error: node %s is offline", nodeName)
			return
		}
	}
}
