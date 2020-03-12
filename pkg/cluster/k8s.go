package cluster

import (
	"context"
	"errors"
	"os"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"

	"k8s.io/client-go/tools/clientcmd"
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
		return k.setUpTiDBCluster(ctx, s)
	case *binlog.ClusterRecommendation:
		return k.setUpBinlogCluster(ctx, s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown(ctx context.Context, spec interface{}) error {
	switch s := spec.(type) {
	case *tidb.TiDBClusterRecommendation:
		return k.TestCli.TiDB.DeleteTiDBCluster(s.TidbCluster)
	default:
		return errors.New("unreachable")
	}
}

// TODO: move the set up process into tidb package and make it a interface
func (k *K8sProvisioner) setUpTiDBCluster(ctx context.Context, recommend *tidb.TiDBClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
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

	// TODO: use ctx for wait end
	err = k.TestCli.TiDB.WaitTiDBClusterReady(recommend.TidbCluster, 10*time.Minute)
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

func (k *K8sProvisioner) setUpBinlogCluster(ctx context.Context, recommend *binlog.ClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
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

func getTiDBNodePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 4000 {
			return port.NodePort
		}
	}
	return 0
}

func getNodeIP(nodeList *corev1.NodeList) string {
	if len(nodeList.Items) == 0 {
		return ""
	}
	return nodeList.Items[0].Status.Addresses[0].Address
}
