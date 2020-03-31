package cluster

import (
	"context"
	"errors"
	"regexp"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
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
	if err := k.CreateNamespace(spec.Cluster.Namespace()); err != nil {
		return nil, nil, errors.New("failed to create namespace " + spec.Cluster.Namespace())
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
func (k *K8sProvisioner) TearDown(ctx context.Context, spec clusterTypes.ClusterSpecs) error {
	if !fixture.Context.Purge {
		return nil
	}
	return spec.Cluster.Delete()
}

// hasIOChaos checks if there is any io chaos nemesis
func (k *K8sProvisioner) hasIOChaos(ngs []string) string {
	for _, v := range ngs {
		switch v {
		case "delay_tikv", "errno_tikv", "mixed_tikv", "readerr_tikv":
			return "tikv"
		case "delay_pd", "errno_pd", "mixed_pd":
			return "pd"
		case "delay_tiflash", "errno_tiflash", "mixed_tiflash", "readerr_tiflash":
			return "tiflash"
		}
	}
	return ""
}

// TODO(yeya24): support other cluster types
func (k *K8sProvisioner) updateClusterDef(spec clusterTypes.ClusterSpecs, ioChaosType string) {
	var tc *v1alpha1.TidbCluster
	switch s := spec.Defs.(type) {
	case *tidb.Recommendation:
		tc = s.TidbCluster
	case *tiflash.Recommendation:
		if ioChaosType == "tiflash" {
			s.TiFlash.StatefulSet.Spec.Template.Annotations = map[string]string{
				"admission-webhook.pingcap.com/request": "chaosfs-tiflash",
			}
			return
		}
		tc = s.TiDBCluster.TidbCluster
	default:
		return
	}

	if ioChaosType == "tikv" {
		tc.Spec.TiKV.Annotations = map[string]string{
			"admission-webhook.pingcap.com/request": "chaosfs-tikv",
		}
	} else if ioChaosType == "pd" {
		tc.Spec.PD.Annotations = map[string]string{
			"admission-webhook.pingcap.com/request": "chaosfs-pd",
		}
	}
}
