package conn_pools

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
)

const (
	// context key
	RequestID = "requestID"
	Timeout   = "timeout"

	// connection pools configuration
	poolSize int32 = 64

	// zookeeper timeout configuration
	zkTimeout = 5 * time.Hour

	// default socket timeout
	socketTimeout = 500 * time.Millisecond
)

var (
	pools   = sync.Map{}
	zkHosts = []string{}
	reqId   = time.Now().Unix()
)

// Init: pools initialization
func Init(zkAddrs string, services map[string]string) {
	if zkAddrs == "" {
		logs.Error("error: zk address not specified, default set to 127.0.0.1:2181")
		zkHosts = []string{"127.0.0.1:2181"}
	} else {
		zkHosts = strings.Split(zkAddrs, ",")
	}

	for k, v := range services {
		ConnPoolAdd(k, v)
	}
	start()
}

// ConnPoolAdd: add pool
func ConnPoolAdd(serviceName string, path string) {
	if _, loaded := pools.Load(serviceName); loaded {
		logs.Info("pool %s already exist", serviceName)
		return
	}
	go func() {
		const (
			statusConnecting = int8(0)
			statusList       = int8(1)
			statusWatch      = int8(2)
		)
		var (
			zkConn            *zk.Conn
			err               error
			status            int8
			eventChan         <-chan zk.Event
			children          []string
			quitChan          chan struct{}
			reconnectInterval = 2 * time.Second
			nextInterval      = reconnectInterval
		)

		for {
			if status == statusConnecting && zkConn == nil {
				zkConn, eventChan, err = zk.Connect(zkHosts, zkTimeout)
				quitChan = make(chan struct{})
			} else if status == statusList {
				children, _, err = zkConn.Children(path)
				if err == nil {
					syncChildren(serviceName, path, children, children)
					status = statusWatch
					logs.Info("pool %v added, nodes:%v", serviceName, children)
					if p, loaded := pools.Load(serviceName); loaded {
						pool, ok := p.(*pool)
						if ok {
							quitChan = pool.quitChan
							// logs.Info("load pool %s successful", serviceName)
						} else {
							logs.Error("error: value of pool %s in pools is %v", serviceName, p)
						}
					} else {
						logs.Error("error: load pool %s failed", serviceName)
					}
					continue
				} else if err.Error() == zk.ErrNoNode.Error() {
					time.Sleep(5 * reconnectInterval)
					continue
				}
			} else if status == statusWatch {
				_, _, eventChan, err = zkConn.ChildrenW(path)
			}

			if err != nil && err.Error() != zk.ErrNoNode.Error() {
				if zkConn != nil {
					zkConn.Close()
					zkConn = nil
				}
				status = statusConnecting
				logs.Error("zk connection error: %v", err)
				time.Sleep(nextInterval)
				nextInterval *= 2
				continue
			}

			loop := true
			for loop {
				select {
				case e := <-eventChan:
					if e.Type == zk.EventSession && e.State == zk.StateConnecting {
						continue
					} else if e.Type == zk.EventSession && e.State == zk.StateConnected {
						status = statusList
						loop = false
						continue
					} else if e.Type == zk.EventNodeChildrenChanged {
						status = statusList
						loop = false
						continue
					} else if e.State == zk.StateDisconnected || e.State == zk.StateExpired {
						zkConn.Close()
						zkConn = nil
						status = statusConnecting
						loop = false
						continue
					}
				case <-time.After(15 * reconnectInterval):
					if status != statusWatch {
						zkConn.Close()
						zkConn = nil
						status = statusConnecting
						loop = false
						logs.Info("no event caught, reconnecting")
					}
				case <-quitChan:
					zkConn.Close()
					logs.Info("pool %v->%v watch goroutine exiting", serviceName, path)
					return
				}
			}
		}
	}()
}

// ConnPoolDel: delete pool
func ConnPoolDel(serviceName string) {
	p, found := pools.Load(serviceName)
	if found {
		pool, ok := p.(*pool)
		if ok {
			close(pool.quitChan)
		} else {
			logs.Error("error: value of pool %s in pools is %v", serviceName, p)
		}
		pools.Delete(serviceName)
		logs.Info("pool %s deleted: %s->%v", serviceName, serviceName, pool.value)
	} else {
		logs.Error("error: pool %v not found", serviceName)
	}
}

// RemoteCall: normal remote call
func RemoteCall(ctx context.Context, serviceName string, clientCallback, funcCallback interface{}, args ...interface{}) (interface{}, error) {
	defer func() {
		if err := recover(); err != nil {
			PrintStack(err)
		}
	}()

	beginTime := time.Now()
	requestID := ""
	timeout := socketTimeout

	var cancel context.CancelFunc
	if ctx != nil {
		if ctx.Value(RequestID) != nil {
			requestID = fmt.Sprint(ctx.Value(RequestID))
		} else {
			id := atomic.AddInt64(&reqId, 1)
			requestID = fmt.Sprint(id)
		}
		if ctx.Value(Timeout) != nil {
			t, ok := ctx.Value(Timeout).(time.Duration)
			if ok {
				timeout = t
			} else {
				logs.Error("error: value of timeout in ctx is not type time.Duration")
			}
		}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	var err error
	var node *node
	var pool *pool
	var conn *thriftClient
	var resv []reflect.Value

	pool, node, err = nodesSelect(serviceName, weightedRoundRobin)
	if err != nil {
		return nil, err
	}

	callThriftFunc := func() error {
		// logs.Info("%v|%v|%v|%v|RemoteCall begin: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), "")
		resv = call(ctx, clientCallback, funcCallback, conn.client, args...)
		if len(resv) == 0 {
			pool.fuseCtrl.onReject()
			node.weightCtrl.onReject()
			node.fuseCtrl.onReject()
			if err := conn.close(); err != nil {
				logs.Error(err)
			}
			err := errors.New("resp is nil: context timeout execeeded")
			logs.Error("%v|%v|%v|%v|RemoteCall error: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), err)
			return err
		}
		e := resv[len(resv)-1].Interface()
		if e != nil {
			pool.fuseCtrl.onReject()
			node.weightCtrl.onReject()
			node.fuseCtrl.onReject()
			if err := conn.close(); err != nil {
				logs.Error(err)
			}
			logs.Error("%v|%v|%v|%v|RemoteCall error: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), e)
			return fmt.Errorf("%v", e)
		}
		pool.fuseCtrl.onSuccess()
		node.weightCtrl.onSuccess()
		node.fuseCtrl.onSuccess()
		putConn(node, conn)
		// logs.Info("%v|%v|%v|%v|RemoteCall review: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), "")
		return nil
	}

	ok := false
	select {
	case conn, ok = <-node.poolChan:
		if !ok {
			logs.Info("%v|%v|%v|node is offline", requestID, serviceName, node.name)
			return nil, errors.New("node is offline")
		}
		if err := callThriftFunc(); err != nil {
			return nil, err
		}
	default:
		conn, err = newThriftClient(node.value, timeout)
		if err != nil {
			pool.fuseCtrl.onReject()
			node.weightCtrl.onReject()
			node.fuseCtrl.onReject()
			return nil, err
		}
		logs.Info("%v|%v|%v|using new client", requestID, serviceName, conn.address)
		if err := callThriftFunc(); err != nil {
			return nil, err
		}
	}

	return resv[0].Interface(), nil
}

// RemoteCallInParallel: query same node in parallel
func RemoteCallInParallel(ctx context.Context, serviceName string, clientCallback, funcCallback, arg interface{}, goNum int) ([]interface{}, error) {
	defer func() {
		if err := recover(); err != nil {
			PrintStack(err)
		}
	}()

	args := make([]interface{}, goNum)
	argValue := reflect.ValueOf(arg)
	if reflect.TypeOf(arg).Kind() == reflect.Slice {
		for i := 0; i < goNum; i++ {
			args[i] = argValue.Index(i).Interface()
		}
	} else {
		return nil, errors.New("arg is not kind of reflect.Slice")
	}

	beginTime := time.Now()
	requestID := ""
	timeout := socketTimeout

	var cancel context.CancelFunc
	if ctx != nil {
		if ctx.Value(requestID) != nil {
			requestID = fmt.Sprint(ctx.Value(requestID))
		} else {
			id := atomic.AddInt64(&reqId, 1)
			requestID = fmt.Sprint(id)
		}
		if ctx.Value(Timeout) != nil {
			t, ok := ctx.Value(Timeout).(time.Duration)
			if ok {
				timeout = t
			} else {
				logs.Error("error: value of timeout in ctx is not type time.Duration")
			}
		}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	var err error
	var node *node
	var pool *pool
	pool, node, err = nodesSelect(serviceName, weightedRoundRobin)
	if err != nil {
		return nil, err
	}
	callThriftFunc := func(conn *thriftClient, arg interface{}) (interface{}, error) {
		// logs.Info("%v|%v|%v|%v|RemoteCall begin: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), "")
		resv := call(ctx, clientCallback, funcCallback, conn.client, arg)
		if len(resv) == 0 {
			pool.fuseCtrl.onReject()
			node.weightCtrl.onReject()
			node.fuseCtrl.onReject()
			if err := conn.close(); err != nil {
				logs.Error(err)
			}
			err := errors.New("resp is nil: context timeout execeeded")
			logs.Error("%v|%v|%v|%v|RemoteCall error: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), err)
			return nil, err
		}
		e := resv[len(resv)-1].Interface()
		if e != nil {
			pool.fuseCtrl.onReject()
			node.weightCtrl.onReject()
			node.fuseCtrl.onReject()
			if err := conn.close(); err != nil {
				logs.Error(err)
			}
			logs.Error("%v|%v|%v|%v|RemoteCall error: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), e)
			return nil, fmt.Errorf("%v", e)
		}
		pool.fuseCtrl.onSuccess()
		node.weightCtrl.onSuccess()
		node.fuseCtrl.onSuccess()
		putConn(node, conn)
		// logs.Info("%v|%v|%v|%v|RemoteCall review: %v|%v", requestID, serviceName, conn.address, time.Since(beginTime), runtime.FuncForPC(reflect.ValueOf(funcCallback).Pointer()).Name(), "")

		return resv[0].Interface(), nil
	}
	callRemote := func(arg interface{}) (interface{}, error) {
		select {
		case conn, ok := <-node.poolChan:
			if !ok {
				logs.Info("%v|%v|%v|node is offline", requestID, serviceName, node.name)
				return nil, errors.New("node is offline")
			}
			return callThriftFunc(conn, arg)
		default:
			conn, err := newThriftClient(node.value, timeout)
			if err != nil {
				pool.fuseCtrl.onReject()
				node.weightCtrl.onReject()
				node.fuseCtrl.onReject()
				return nil, err
			}
			logs.Info("%v|%v|%v|using new client", requestID, serviceName, conn.address)
			return callThriftFunc(conn, arg)
		}
	}

	resps := make([]interface{}, goNum)
	errs := make([]error, goNum)
	wg := sync.WaitGroup{}
	wg.Add(goNum)
	for i := 0; i < goNum; i++ {
		go func(i int) {
			defer func() {
				wg.Done()
				if err := recover(); err != nil {
					PrintStack(err)
				}
			}()
			rsp, err := callRemote(args[i])
			resps[i] = rsp
			if err != nil {
				errs[i] = err
			}
		}(i)
	}
	wg.Wait()
	errStrs := make([]string, 0, goNum)
	for _, err := range errs {
		if err != nil {
			errStrs = append(errStrs, err.Error())
		}
	}
	if len(errStrs) > 0 {
		return nil, errors.New(strings.Join(errStrs, ","))
	}

	return resps, nil
}

func call(ctx context.Context, clientCallback, funcCallback interface{}, client *thrift.TStandardClient, args ...interface{}) []reflect.Value {
	// get client
	cliFunc := reflect.ValueOf(clientCallback)
	cliArgs := make([]reflect.Value, 1)
	cliArgs[0] = reflect.ValueOf(client)
	rCli := cliFunc.Call(cliArgs)

	// get result
	fFunc := reflect.ValueOf(funcCallback)
	fArgs := make([]reflect.Value, len(args)+2)
	fArgs[0] = rCli[0]
	fArgs[1] = reflect.ValueOf(ctx)
	for i, arg := range args {
		fArgs[i+2] = reflect.ValueOf(arg)
	}

	respChan := make(chan []reflect.Value, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				PrintStack(err)
			}
		}()
		respChan <- fFunc.Call(fArgs)
	}()
	select {
	case resp := <-respChan:
		return resp
	case <-ctx.Done():
		return nil
	}
}
