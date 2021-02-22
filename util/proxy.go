package util

import (
	"context"
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/go-sql-driver/mysql"
	netproxy "golang.org/x/net/proxy"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

var (
	setMySQLProxyOnce sync.Once
)

// SetMySQLProxy sets the mysql dial specified by the proxy-related variables in the environment,
// and makes underlying connections directly.
// ALL_PROXY/all_proxy, NO_PROXY/no_proxy
func SetMySQLProxy() {
	dialer := fromContextFlag()

	setMySQLProxyOnce.Do(func() {
		mysql.RegisterDialContext("tcp", func(ctx context.Context, addr string) (net.Conn, error) {
			if xd, ok := dialer.(netproxy.ContextDialer); ok {
				return xd.DialContext(ctx, "tcp", addr)
			}
			return dialer.Dial("tcp", addr)
		})
	})
}

var (
	kubeProxyEnv = &envOnce{
		names: []string{"KUBE_PROXY", "kube_proxy"},
	}
)

// envOnce looks up an environment variable (optionally by multiple
// names) once. It mitigates expensive lookups on some platforms
// (e.g. Windows).
// (Borrowed from net/proxy/proxy.go)
type envOnce struct {
	names []string
	once  sync.Once
	val   string
}

func (e *envOnce) Get() string {
	e.once.Do(e.init)
	return e.val
}

func (e *envOnce) init() {
	for _, n := range e.names {
		e.val = os.Getenv(n)
		if e.val != "" {
			return
		}
	}
}

func fromEnvironment() netproxy.Dialer {
	direct := &net.Dialer{}
	kubeProxy := kubeProxyEnv.Get()

	if len(kubeProxy) == 0 {
		return direct
	}
	proxyURL, err := url.Parse(kubeProxy)

	if err != nil {
		return direct
	}
	proxy, err := netproxy.FromURL(proxyURL, direct)
	if err != nil {
		return direct
	}
	return proxy
}

func fromContextFlag() netproxy.Dialer {
	direct := &net.Dialer{}
	if fixture.Context.K8sProxy == "" {
		return direct
	}

	proxyURL, err := url.Parse(fixture.Context.K8sProxy)

	if err != nil {
		return direct
	}
	proxy, err := netproxy.FromURL(proxyURL, direct)
	if err != nil {
		return direct
	}
	return proxy
}
