package util

import (
	"context"
	"net"
	"net/url"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/ngaut/log"
	netproxy "golang.org/x/net/proxy"
)

var (
	setMySQLProxyOnce sync.Once
)

// SetMySQLProxy sets the proxy mysql dial specified in the command flag,
// and makes underlying connections directly.
func SetMySQLProxy(proxyURL string) {
	if proxyURL == "" {
		return
	}
	dialer := fromURL(proxyURL)

	setMySQLProxyOnce.Do(func() {
		mysql.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			if xd, ok := dialer.(netproxy.ContextDialer); ok {
				return xd.DialContext(ctx, "tcp", addr)
			}
			return dialer.Dial("tcp", addr)
		})
	})
}

func fromURL(proxyStr string) netproxy.Dialer {
	direct := &net.Dialer{}
	proxyURL, err := url.Parse(proxyStr)

	if err != nil {
		log.Warnf("parse proxy URL error, fallback to direct dialer: %v", err)
		return direct
	}
	proxy, err := netproxy.FromURL(proxyURL, direct)
	if err != nil {
		return direct
	}
	return proxy
}
