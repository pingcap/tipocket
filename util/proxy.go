package util

import (
	"context"
	"net"
	"sync"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/net/proxy"
)

var (
	setMySQLProxyOnce sync.Once
)

// SetMySQLProxyFromEnvironment sets the mysql dial specified by the proxy-related variables in the environment,
// and makes underlying connections directly.
// ALL_PROXY/all_proxy, NO_PROXY/no_proxy
func SetMySQLProxyFromEnvironment() {
	dialer := proxy.FromEnvironment()

	setMySQLProxyOnce.Do(func() {
		mysql.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer.Dial("tcp", addr)
		})
	})
}
