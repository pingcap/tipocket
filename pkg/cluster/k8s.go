package cluster

import (
	"context"
	"errors"
	"regexp"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/abtest"
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
func NewK8sProvisioner() clusterTypes.Provisioner {
	return &K8sProvisioner{
		TestCli: tests.TestClient,
	}
}

// SetUp sets up cluster, returns err or all nodes info
func (k *K8sProvisioner) SetUp(ctx context.Context, spec interface{}) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	switch s := spec.(type) {
	case *tidb.Recommendation:
		return k.setUpTiDBCluster(s)
	case *binlog.Recommendation:
		return k.setUpBinlogCluster(s)
	case *abtest.Recommendation:
		return k.setUpABTestCluster(s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown(ctx context.Context, spec interface{}) error {
	var err error
	if !fixture.Context.Purge {
		return nil
	}
	switch s := spec.(type) {
	case *tidb.Recommendation:
		err = k.tearDownTiDBCluster(s)
	case *binlog.Recommendation:
		err = k.tearDownBinlogCluster(s)
	case *abtest.Recommendation:
		err = k.tearDownABtestCluster(s)
	default:
		return errors.New("unreachable")
	}
	return err
}

// TODO: move the set up process into tidb package and make it a interface
func (k *K8sProvisioner) setUpTiDBCluster(recommend *tidb.Recommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	if tc, err := k.TiDB.GetTiDBCluster(recommend.NS, recommend.NS); err == nil {
		recommend.TidbCluster = tc
	} else {
		err = k.TiDB.ApplyTiDBCluster(recommend)
		if err != nil {
			return nodes, clientNodes, err
		}
	}
	nodes, err = k.TiDB.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.TiDB.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	return nodes, clientNodes, err
}

func (k *K8sProvisioner) setUpBinlogCluster(recommend *binlog.Recommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	err = k.Binlog.Apply(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	nodes, err = k.Binlog.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	clientNodes, err = k.Binlog.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	return nodes, clientNodes, err
}

func (k *K8sProvisioner) setUpABTestCluster(recommend *abtest.Recommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	err = k.ABTest.Apply(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	nodes, err = k.ABTest.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	clientNodes, err = k.ABTest.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	return nodes, clientNodes, err
}

func (k *K8sProvisioner) tearDownTiDBCluster(tc *tidb.Recommendation) error {
	if err := k.TiDB.Delete(tc); err != nil {
		return err
	}
	return k.DeleteNamespace(tc.Namespace)
}

func (k *K8sProvisioner) tearDownBinlogCluster(tc *binlog.Recommendation) error {
	if err := k.Binlog.Delete(tc); err != nil {
		return err
	}
	return k.DeleteNamespace(tc.NS)
}

func (k *K8sProvisioner) tearDownABtestCluster(tc *abtest.Recommendation) error {
	if err := k.ABTest.Delete(tc); err != nil {
		return err
	}
	return k.DeleteNamespace(tc.NS)
}
