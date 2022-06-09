package conn_pools

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/astaxie/beego/logs"
)

func turnOnGracefulExit() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		for s := range ch {
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT:
				logs.Info("exit on signal: ", s)
				gracefulExit()
			case syscall.SIGUSR1:
				logs.Info("get signal usr1: ", s)
			case syscall.SIGUSR2:
				logs.Info("get signal usr2: ", s)
			default:
				logs.Info("get signal: ", s)
			}
		}
	}()
}
