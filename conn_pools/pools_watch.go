package conn_pools

import "sync"

type pool struct {
	sync.RWMutex
	name     string
	value    string
	nodes    []*node
	fuseCtrl fuseControl
	quitChan chan struct{}
}
