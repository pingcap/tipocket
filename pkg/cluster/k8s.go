package cluster

import (
	"context"
	"errors"
	"regexp"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
)

var (
	tidbRegex = regexp.MustCompile(`.*tidb-[0-9]+$`)
)

// K8sProvider implement Provider in k8s
type K8sProvider struct {
	*tests.TestCli
}

// NewK8sClusterProvider create tidb cluster on k8s
func NewK8sClusterProvider() Provider {
	return &K8sProvider{
		TestCli: tests.TestClient,
	}
}

// SetUp sets up cluster, returns err or all nodes info
func (k *K8sProvider) SetUp(_ context.Context, spec Specs) ([]Node, []ClientNode, error) {
	if err := k.CreateNamespace(spec.Namespace); err != nil {
		return nil, nil, errors.New("failed to create namespace " + spec.Namespace + err.Error())
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
func (k *K8sProvider) TearDown(_ context.Context, spec Specs) error {
	if !fixture.Context.Purge {
		return nil
	}
	if fixture.Context.DeleteNS {
		return k.DeleteNamespace(spec.Namespace)
	}
	return spec.Cluster.Delete()
}
