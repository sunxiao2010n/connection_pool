package conn_pools

import (
	"fmt"
)

func nodesSelect(serviceName string, selectMethod func(*pool) (*node, error)) (*pool, *node, error) {
	p, ok := pools.Load(serviceName)
	if !ok {
		return nil, nil, fmt.Errorf("service %s not found", serviceName)
	}

	pool, ok := p.(*pool)
	if !ok {
		return nil, nil, fmt.Errorf("value of pool %s in pools is %v", serviceName, p)
	}
	if !pool.fuseCtrl.available() {
		return nil, nil, fmt.Errorf("pool %s is not available", serviceName)
	}

	pool.RLock()
	length := len(pool.nodes)
	if length == 0 {
		pool.RUnlock()
		return pool, nil, fmt.Errorf("no node of service %s discovered", serviceName)
	}
	pool.RUnlock()

	var fuseCount int
	for {
		n, err := selectMethod(pool)
		if err != nil {
			return pool, nil, err
		}
		if !n.fuseCtrl.available() {
			fuseCount++
			if fuseCount >= length {
				return pool, nil, fmt.Errorf("no node of pool %s is available", pool.name)
			}
			continue
		}

		return pool, n, nil
	}
}
