package cluster

import (
	"context"
	"errors"
	"os"
	"regexp"

	"k8s.io/client-go/tools/clientcmd"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
)

var (
	tidbRegex = regexp.MustCompile(`.*tidb-[0-9]+$`)
)

// K8sProvisioner implement Provisioner in k8s
type K8sProvisioner struct {
	*tests.TestCli
}

// NewK8sProvisioner create k8s provisioner
func NewK8sProvisioner() (clusterTypes.Provisioner, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	return &K8sProvisioner{
		TestCli: tests.NewTestCli(conf),
	}, nil
}

// SetUp sets up cluster, returns err or all nodes info
func (k *K8sProvisioner) SetUp(ctx context.Context, spec interface{}) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	switch s := spec.(type) {
	case *tidb.TiDBClusterRecommendation:
		return k.setUpTiDBCluster(s)
	case *binlog.ClusterRecommendation:
		return k.setUpBinlogCluster(s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown(ctx context.Context, spec interface{}) error {
	switch s := spec.(type) {
	case *tidb.TiDBClusterRecommendation:
		return k.tearDownTiDBCluster(s)
	default:
		return errors.New("unreachable")
	}
}

// TODO: move the set up process into tidb package and make it a interface
func (k *K8sProvisioner) setUpTiDBCluster(recommend *tidb.TiDBClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.TestCli.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	err = k.TestCli.TiDB.ApplyTiDBCluster(recommend.TidbCluster)
	if err != nil {
		return nodes, clientNodes, err
	}

	err = k.TestCli.TiDB.WaitTiDBClusterReady(recommend.TidbCluster, fixture.Context.WaitClusterReadyDuration)
	if err != nil {
		return nodes, clientNodes, err
	}

	err = k.TestCli.TiDB.ApplyTiDBService(recommend.Service)
	if err != nil {
		return nodes, clientNodes, err
	}

	nodes, err = k.TestCli.TiDB.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.TestCli.TiDB.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	return nodes, clientNodes, err
}

func (k *K8sProvisioner) setUpBinlogCluster(recommend *binlog.ClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)

	if err := k.TestCli.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	err = k.TestCli.Binlog.Apply(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	nodes, err = k.TestCli.Binlog.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.TestCli.Binlog.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	return nodes, clientNodes, err
}

func (k *K8sProvisioner) tearDownTiDBCluster(recommend *tidb.TiDBClusterRecommendation) error {
	if err := k.TestCli.TiDB.DeleteTiDBCluster(recommend.TidbCluster); err != nil {
		return err
	}
	if fixture.Context.PurgeNsOnSuccess {
		return k.TestCli.DeleteNamespace(recommend.Name)
	}
	return nil
}
