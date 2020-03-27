package cluster

import (
	"context"
	"errors"
	"regexp"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/abtest"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tiflash"
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
func (k *K8sProvisioner) SetUp(ctx context.Context, spec clusterTypes.ClusterSpecs) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	// update cluster spec if there is an io chaos nemesis
	k.updateClusterDef(spec, k.hasIOChaos(spec.NemesisGens))
	switch s := spec.Defs.(type) {
	case *tidb.Recommendation:
		return k.setUpTiDBCluster(s)
	case *binlog.Recommendation:
		return k.setUpBinlogCluster(s)
	case *abtest.Recommendation:
		return k.setUpABTestCluster(s)
	case *cdc.Recommendation:
		return k.setUpCDCCluster(s)
	case *tiflash.Recommendation:
		return k.setUpTiFlashCluster(s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown(ctx context.Context, spec clusterTypes.ClusterSpecs) error {
	var err error
	if !fixture.Context.Purge {
		return nil
	}
	switch s := spec.Defs.(type) {
	case *tidb.Recommendation:
		err = k.tearDownTiDBCluster(s)
	case *binlog.Recommendation:
		err = k.tearDownBinlogCluster(s)
	case *cdc.Recommendation:
		err = k.tearDownCDCCluster(s)
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

func (k *K8sProvisioner) setUpCDCCluster(recommend *cdc.Recommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	err = k.CDC.Apply(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	nodes, err = k.CDC.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	clientNodes, err = k.CDC.GetClientNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}
	return nodes, clientNodes, err
}

func (k *K8sProvisioner) setUpTiFlashCluster(recommend *tiflash.Recommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	if err := k.CreateNamespace(recommend.NS); err != nil {
		return nil, nil, errors.New("failed to create namespace " + recommend.NS)
	}
	if err = k.TiFlash.Apply(recommend); err != nil {
		return nodes, clientNodes, err
	}

	nodes, err = k.TiFlash.GetNodes(recommend)
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.TiFlash.GetClientNodes(recommend)
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

func (k *K8sProvisioner) tearDownCDCCluster(tc *cdc.Recommendation) error {
	if err := k.CDC.Delete(tc); err != nil {
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

// hasIOChaos checks if there is any io chaos nemesis
func (k *K8sProvisioner) hasIOChaos(ngs []string) string {
	for _, v := range ngs {
		switch v {
		case "delay_tikv", "errno_tikv", "mixed_tikv", "readerr_tikv":
			return "tikv"
		case "delay_pd", "errno_pd", "mixed_pd":
			return "pd"
		}
	}
	return ""
}

// TODO(yeya24): support other cluster types
func (k *K8sProvisioner) updateClusterDef(spec clusterTypes.ClusterSpecs, ioChaosType string) {
	switch s := spec.Defs.(type) {
	case *tidb.Recommendation:
		if ioChaosType == "tikv" {
			s.TidbCluster.Spec.TiKV.Annotations = map[string]string{
				"admission-webhook.pingcap.com/request": "chaosfs-tikv",
			}
		} else if ioChaosType == "pd" {
			s.TidbCluster.Spec.PD.Annotations = map[string]string{
				"admission-webhook.pingcap.com/request": "chaosfs-pd",
			}
		}
	}
}
