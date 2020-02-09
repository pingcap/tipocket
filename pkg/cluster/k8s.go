package cluster

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tests/util"

	"k8s.io/client-go/tools/clientcmd"
)

// K8sProvisioner implement Provisioner in k8s
type K8sProvisioner struct {
	*util.E2eCli
}

// NewK8sProvisioner create k8s provisioner
func NewK8sProvisioner() (Provisioner, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	return &K8sProvisioner{
		E2eCli: util.NewE2eCli(conf),
	}, nil
}

// SetUp sets up cluster, returns err or all nodes info
func (k *K8sProvisioner) SetUp(ctx context.Context, spec interface{}) ([]Node, error) {
	switch s := spec.(type) {
	case *tidb.TiDBClusterRecommendation:
		return k.setUpTiDBCluster(ctx, s)
	default:
		panic("unreachable")
	}
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown() error {
	return nil
}

func (k *K8sProvisioner) setUpTiDBCluster(ctx context.Context, recommand *tidb.TiDBClusterRecommendation) ([]Node, error) {
	var (
		nodes []Node
		err   error
	)

	err = k.E2eCli.TiDB.ApplyTiDBCluster(recommand.TidbCluster)
	if err != nil {
		return nodes, err
	}

	// TODO: use ctx for wait end
	err = k.E2eCli.TiDB.WaitTiDBClusterReady(recommand.TidbCluster, 10*time.Minute)
	if err != nil {
		return nodes, err
	}

	err = k.E2eCli.TiDB.ApplyTiDBService(recommand.Service)
	if err != nil {
		return nodes, err
	}

	pods, err := k.E2eCli.TiDB.GetNodes()
	log.Println(pods, err)

	return nodes, err
}
