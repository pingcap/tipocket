package cluster

import (
	"context"
	"errors"
	"regexp"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
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
func (k *K8sProvisioner) SetUp(_ context.Context, spec clusterTypes.ClusterSpecs) ([]clusterTypes.Node, []clusterTypes.ClientNode, error) {
	if err := k.CreateNamespace(spec.Namespace); err != nil {
		return nil, nil, errors.New("failed to create namespace " + spec.Namespace)
	}
	if err := spec.Cluster.Apply(); err != nil {
		return nil, nil, err
	}
	nodes, err := spec.Cluster.GetNodes()
	if err != nil {
		return nil, nil, err
	}
	clientNodes, err := spec.Cluster.GetClientNodes()
	if err != nil {
		return nil, nil, err
	}
	return nodes, clientNodes, nil
}

// TearDown tears down the cluster
func (k *K8sProvisioner) TearDown(_ context.Context, spec clusterTypes.ClusterSpecs) error {
	if !fixture.Context.Purge {
		return nil
	}
	if fixture.Context.DeleteNS {
		return k.DeleteNamespace(spec.Namespace)
	}
	return spec.Cluster.Delete()
}
