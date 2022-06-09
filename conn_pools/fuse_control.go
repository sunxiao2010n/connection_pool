package conn_pools

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego/logs"
)

// fuse control configuration
const (
	fuseWindow = int64(2)

	poolFuseTime      = int64(5)
	poolFailTimesSucc = float64(2)

	nodeFuseTime      = int64(5)
	nodeFailTimesSucc = float64(2)
)

type fuseControl struct {
	sync.Mutex
	poolName         string
	nodeName         string
	fuse             int32
	lastRecordSecond int64
	failTimes        [fuseWindow]float64
	succTimes        [fuseWindow]float64
	totalFail        float64
	totalSucc        float64
}

func (f *fuseControl) onSuccess() {
	f.Lock()
	defer f.Unlock()
	lastRecordSecond := f.lastRecordSecond
	second := time.Now().Unix()
	index := second % fuseWindow
	if lastRecordSecond != second {
		for flushCounter := int64(1); (lastRecordSecond+flushCounter) <= second && flushCounter <= fuseWindow; flushCounter++ {
			flushIdx := (lastRecordSecond + flushCounter) % fuseWindow
			f.totalFail -= f.failTimes[flushIdx]
			f.totalSucc -= f.succTimes[flushIdx]
			f.failTimes[flushIdx] = f.failTimes[(flushIdx+1)%fuseWindow]
			f.succTimes[flushIdx] = f.succTimes[(flushIdx+1)%fuseWindow]
		}
		f.failTimes[index] = 0
		f.succTimes[index] = 0
		f.lastRecordSecond = second
	}
	f.succTimes[index]++
	f.totalSucc++
}

func (f *fuseControl) onReject() {
	f.Lock()
	defer f.Unlock()
	lastRecordSecond := f.lastRecordSecond
	second := time.Now().Unix()
	index := second % fuseWindow
	if lastRecordSecond != second {
		for flushCounter := int64(1); (lastRecordSecond+flushCounter) <= second && flushCounter <= fuseWindow; flushCounter++ {
			flushIdx := (lastRecordSecond + flushCounter) % fuseWindow
			f.totalFail -= f.failTimes[flushIdx]
			f.totalSucc -= f.succTimes[flushIdx]
			f.failTimes[flushIdx] = f.failTimes[(flushIdx+1)%fuseWindow]
			f.succTimes[flushIdx] = f.succTimes[(flushIdx+1)%fuseWindow]
		}
		f.failTimes[index] = 0
		f.succTimes[index] = 0
		f.lastRecordSecond = second
	}
	f.failTimes[index]++
	f.totalFail++

	if f.nodeName == "" && f.totalFail > poolFailTimesSucc*f.totalSucc {
		if p, loaded := pools.Load(f.poolName); loaded {
			pool, ok := p.(*pool)
			if ok && f.totalFail > float64(int64(len(pool.nodes))*fuseWindow) {
				f.fusing()
			} else if !ok {
				logs.Error("error: value of pool %s in pools is %v", f.poolName, p)
			}
		} else {
			logs.Error("error: load pool %s failed", f.poolName)
		}
	} else if f.nodeName != "" && f.totalFail > nodeFailTimesSucc*f.totalSucc {
		if f.totalFail > float64(fuseWindow) {
			f.fusing()
		}
	}
}

func (f *fuseControl) available() bool {
	if atomic.LoadInt32(&f.fuse) == 1 {
		second := time.Now().Unix()
		lastRecordSecond := atomic.LoadInt64(&f.lastRecordSecond)
		if f.nodeName == "" && second > lastRecordSecond && second-lastRecordSecond > poolFuseTime {
			atomic.StoreInt32(&f.fuse, 0)
			return true
		} else if f.nodeName != "" && second > lastRecordSecond && second-lastRecordSecond > nodeFuseTime {
			atomic.StoreInt32(&f.fuse, 0)
			return true
		}
		return false
	} else {
		return true
	}
}

func (f *fuseControl) fusing() {
	if atomic.SwapInt32(&f.fuse, 1) == 1 {
		return
	}
	p, found := pools.Load(f.poolName)
	if found {
		pool, ok := p.(*pool)
		if !ok {
			logs.Error("error: value of pool %s in pools is %v", f.poolName, p)
		}
		pool.RLock()
		defer pool.RUnlock()
	OUTER:
		for _, node := range pool.nodes {
			connCount := 0
			loop := true
			if node.name == f.nodeName {
				select {
				case c := <-node.poolChan:
					if err := c.close(); err != nil {
						logs.Error(err)
					}
					connCount++
				default:
					loop = false
				}
				logs.Error("error: node %v in pool %v is fusing", f.nodeName, f.poolName)
				break OUTER
			}
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
		}
		if f.nodeName == "" {
			logs.Error("error: pool %v is fusing", f.poolName)
		}
	} else {
		logs.Error("error: pool %s not found", f.poolName)
	}
}
