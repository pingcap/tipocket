package raftstorecheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/debugpb"
	"go.etcd.io/etcd/pkg/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// SecurityConfig is the configuration for supporting tls.
type SecurityConfig struct {
	// CAPath is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	CAPath string `toml:"cacert-path" json:"cacert-path"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	CertPath string `toml:"cert-path" json:"cert-path"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	KeyPath string `toml:"key-path" json:"key-path"`
	// CertAllowedCN is a CN which must be provided by a client
	CertAllowedCN []string `toml:"cert-allowed-cn" json:"cert-allowed-cn"`
}

// ToTLSConfig generates tls config.
func (s SecurityConfig) ToTLSConfig() (*tls.Config, error) {
	if len(s.CertPath) == 0 && len(s.KeyPath) == 0 {
		return nil, nil
	}
	allowedCN, err := s.GetOneAllowedCN()
	if err != nil {
		return nil, err
	}

	tlsInfo := transport.TLSInfo{
		CertFile:      s.CertPath,
		KeyFile:       s.KeyPath,
		TrustedCAFile: s.CAPath,
		AllowedCN:     allowedCN,
	}

	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return tlsConfig, nil
}

// GetOneAllowedCN only gets the first one CN.
func (s SecurityConfig) GetOneAllowedCN() (string, error) {
	switch len(s.CertAllowedCN) {
	case 1:
		return s.CertAllowedCN[0], nil
	case 0:
		return "", nil
	default:
		return "", errors.New("Currently only supports one CN")
	}
}

// GetClientConn returns a gRPC client connection.
// creates a client connection to the given target. By default, it's
// a non-blocking dial (the function won't wait for connections to be
// established, and connecting happens in the background). To make it a blocking
// dial, use WithBlock() dial option.
//
// In the non-blocking case, the ctx does not act against the connection. It
// only controls the setup steps.
//
// In the blocking case, ctx can be used to cancel or expire the pending
// connection. Once this function returns, the cancellation and expiration of
// ctx will be noop. Users should call ClientConn.Close to terminate all the
// pending operations after this function returns.
func GetClientConn(ctx context.Context, addr string, tlsCfg *tls.Config, do ...grpc.DialOption) (*grpc.ClientConn, error) {
	opt := grpc.WithInsecure()
	if tlsCfg != nil {
		creds := credentials.NewTLS(tlsCfg)
		opt = grpc.WithTransportCredentials(creds)
	}
	u, err := url.Parse(addr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cc, err := grpc.DialContext(ctx, u.Host, append(do, opt)...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cc, nil
}

// baseClient is a basic client for all other complex client.
type TiKvDebugClients struct {
	urls        []string
	clientConns map[string]*grpc.ClientConn

	debugClients map[string]debugpb.DebugClient

	ctx    context.Context
	cancel context.CancelFunc

	security SecurityOption

	gRPCDialOptions []grpc.DialOption
	timeout         time.Duration
}

// SecurityOption records options about tls
type SecurityOption struct {
	CAPath   string
	CertPath string
	KeyPath  string
}

// newBaseClient returns a new baseClient.
func NewTiKvDebugClient(ctx context.Context, urls []string /*, security SecurityOption*/) (*TiKvDebugClients, error) {
	ctx1, cancel := context.WithCancel(ctx)
	c := &TiKvDebugClients{
		urls:   urls,
		ctx:    ctx1,
		cancel: cancel,
		//security: security,
	}
	c.debugClients = make(map[string]debugpb.DebugClient)
	for i := 0; i < len(urls); i++ {
		if err := c.AddDebugClient(urls[i]); err != nil {
			log.Fatalf("create tikv debug client error: %v", err)
		}
	}

	return c, nil
}

func (c *TiKvDebugClients) AddDebugClient(addr string) error {
	_, ok := c.debugClients[addr]
	if ok {
		return errors.New("already has same address")
	}
	/*tlsCfg, err := SecurityConfig{
		CAPath:   c.security.CAPath,
		CertPath: c.security.CertPath,
		KeyPath:  c.security.KeyPath,
	}.ToTLSConfig()
	if err != nil {
		return errors.WithStack(err)
	}*/
	dctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	addr = fmt.Sprintf("http://%s:20160", addr)
	log.Infof("connect tikv debug client url %s", addr)
	cc, err := GetClientConn(dctx, addr, nil, c.gRPCDialOptions...)
	if err != nil {
		return errors.WithStack(err)
	}

	c.debugClients[addr] = debugpb.NewDebugClient(cc)
	return nil
}

type CheckRegionCollector struct {
	State        *debugpb.PeerCurrentState
	Report_peers map[uint64]bool
}

func (c *TiKvDebugClients) CheckRaftStoreConsistency() {
	log.Infof("start checking raftstore consistency")

	regionCollector := make(map[uint64]*CheckRegionCollector)
	peerToState := make(map[uint64]*debugpb.PeerCurrentState)

	for url, debugClient := range c.debugClients {
		request := &debugpb.CollectPeerCurrentStateRequest{
			TimeoutSecs: 120,
		}
		response, err := debugClient.CollectPeerCurrentState(c.ctx, request)
		if err != nil {
			log.Warnf("%s collect peer current state err %v", url, err)
			return
		}
		log.Infof("%s collect %v num peer state", url, len(response.States))
		for i := 0; i < len(response.States); i++ {
			currentState := response.States[i]
			if !currentState.Valid {
				log.Warnf("region %v on %s can not get current state", currentState.RegionId, url)
				continue
			}
			logHeader := fmt.Sprintf("region %v on %s current state %v", currentState.RegionId, url, currentState)
			collector, ok := regionCollector[currentState.RegionId]
			if !ok {
				collector := &CheckRegionCollector{
					State:        currentState,
					Report_peers: make(map[uint64]bool),
				}
				collector.Report_peers[currentState.PeerId] = true
				regionCollector[currentState.RegionId] = collector
				continue
			}
			if collector.State.Region != currentState.Region ||
				collector.State.LeaderId != currentState.LeaderId ||
				collector.State.LastIndex != currentState.LastIndex ||
				collector.State.AppliedIndex != currentState.AppliedIndex {
				log.Warnf("not match to origin state, %s, origin state %v", logHeader, collector.State)
				continue
			}
			if _, ok = collector.Report_peers[currentState.PeerId]; ok {
				log.Warnf("peer id is equal to another peer in the same region, %s, origin state %v", logHeader, collector.State)
				continue
			}
			if originState, ok := peerToState[currentState.PeerId]; ok {
				log.Warnf("peer id is equal to another peer in the different region state, %s, origin state %v", logHeader, originState)
				continue
			}
		}
	}

	/*var rangeRegions []*debugpb.PeerCurrentState
	for region_id, state := range region_collector {

	}*/

	log.Infof("end checking raftstore consistency")
}
