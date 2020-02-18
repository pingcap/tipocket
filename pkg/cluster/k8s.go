package cluster

import (
	"context"
	"os"
	"regexp"
	"time"

	corev1 "k8s.io/api/core/v1"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tests/util"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	tidbRegex = regexp.MustCompile(`.*tidb-[0-9]+$`)
)

// K8sProvisioner implement Provisioner in k8s
type K8sProvisioner struct {
	*util.E2eCli
}

// NewK8sProvisioner create k8s provisioner
func NewK8sProvisioner() (clusterTypes.Provisioner, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	return &K8sProvisioner{
		E2eCli: util.NewE2eCli(conf),
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
func (k *K8sProvisioner) TearDown() error {
	return nil
}

// TODO: move the set up process into tidb package and make it a interface
func (k *K8sProvisioner) setUpTiDBCluster(ctx context.Context, recommand *tidb.TiDBClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)
	err = k.E2eCli.TiDB.ApplyTiDBCluster(recommend.TidbCluster)
	if err != nil {
		return nodes, clientNodes, err
	}

	// TODO: use ctx for wait end
	err = k.E2eCli.TiDB.WaitTiDBClusterReady(recommend.TidbCluster, 10*time.Minute)
	if err != nil {
		return nodes, clientNodes, err
	}

	err = k.E2eCli.TiDB.ApplyTiDBService(recommend.Service)
	if err != nil {
		return nodes, clientNodes, err
	}

	nodes, err = k.E2eCli.TiDB.GetNodes(recommand)
	if err != nil {
		return nodes, clientNodes, err
	}

	k8sNodes, err := k.E2eCli.GetNodes()
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.E2eCli.TiDB.GetClientNodes(recommand)
	if err != nil {
		return nodes, clientNodes, err
	}

	return nodes, clientNodes, err
}

func (k *K8sProvisioner) setUpBinlogCluster(ctx context.Context, recommand *binlog.ClusterRecommendation) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	var (
		nodes       []clusterTypes.Node
		clientNodes []clusterTypes.ClientNode
		err         error
	)

	err = k.E2eCli.Binlog.Apply(recommand)
	if err != nil {
		return nodes, clientNodes, err
	}

	nodes, err = k.E2eCli.Binlog.GetNodes(recommand)
	if err != nil {
		return nodes, clientNodes, err
	}

	clientNodes, err = k.E2eCli.Binlog.GetClientNodes(recommand)
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
