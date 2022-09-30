package zk_tool

import (
	"errors"
	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
	"path"
	"strings"
)

type ZkTool struct {
	ZKServers string
	conn      *zk.Conn
}

func (zTool *ZkTool) Init(zkAddr string) {
	zTool.ZKServers = zkAddr
	zTool.connect()
}

func (zTool *ZkTool) connect() error {
	servers := strings.Split(zTool.ZKServers, ",")
	//注意，第二个参数为创建之后session维持的时间，因为session消
	//失之后，才会将注册的ip地址从zk上摘除，所以不能太长，否则影响服务
	//正常功能，一般为1s.
	conn, _, err := zk.Connect(servers, 1000000) //测试所以写的60s
	zTool.conn = conn
	return err
}

func (zTool *ZkTool) CreateZPath(zPath string, zkFlags int32) error {
	if zTool.conn == nil {
		zTool.connect()
		logs.Info("zk connected\n")
	}
	conn := zTool.conn
	logs.Info("ZkTool: creating: %v, %v\n", zPath, zkFlags)
	_, err := conn.Create(zPath, nil, zkFlags, zk.WorldACL(zk.PermAll))
	if err != nil {
		if zk.ErrNodeExists == err {
			logs.Info("ZkTool: node exists: %v\n", zPath)
			return nil
		} else {
			logs.Info("ZkTool: not created")
			parent, _ := path.Split(zPath)
			if len(parent) == 0 {
				return errors.New("path is blank")
			}
			c := parent[:len(parent)-1]
			logs.Info("ZkTool: creating : %v, seq\n", c)
			err = zTool.CreateZPath(c, zk.FlagSequence)
			if err != nil {
				return err
			}
			logs.Info("ZkTool: finally, creating : %v, %v\n", zPath, zkFlags)
			_, err = conn.Create(zPath, nil, zkFlags, zk.WorldACL(zk.PermAll))
			if err == zk.ErrNodeExists {
				err = nil
			}
		}
	}

	if zk.ErrNodeExists == err {
		return nil
	} else {
		return err
	}
}
