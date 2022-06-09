package conn_pools

import (
	"errors"
	"fmt"
	"net"
	"runtime"

	"github.com/astaxie/beego/logs"
)

// LocalIP: get one valid local ip
func LocalIP() (net.IP, error) {
	getIpFromAddr := func(addr net.Addr) net.IP {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			return nil
		}
		ip = ip.To4()
		if ip == nil {
			return nil // not an ipv4 address
		}

		return ip
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("ip address not found, please check NICs")
}

// PrintStack: log stack and return stack
func PrintStack(err interface{}, args ...interface{}) string {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	errStr := fmt.Sprintf("%+v\n%+v\n%+v", args, err, string(buf[:n]))
	logs.Error("recover error==>\n%s", errStr)

	return errStr
}
