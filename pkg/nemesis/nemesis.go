package nemesis

import (
	"context"
	"log"
	"os"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/util/net"
)

type kill struct{}

func (kill) Invoke(ctx context.Context, node cluster.Node, chaosNS string, args ...string) error {
	c, err := createClient()
	if err != nil {
		return err
	}
	log.Printf("Creating pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	return podChaos(ctx, c, chaosNS, node.Namespace, node.PodName, v1alpha1.PodKillAction)
}

func (kill) Recover(ctx context.Context, node cluster.Node, chaosNS string, args ...string) error {
	c, err := createClient()
	if err != nil {
		return err
	}
	log.Printf("Recover pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	return cancelPodChaos(ctx, c, chaosNS, node.Namespace, node.PodName, v1alpha1.PodKillAction)
}

func (kill) Name() string {
	return string(core.PodFailure)
}

type drop struct {
	t net.IPTables
}

func (n drop) Invoke(ctx context.Context, node cluster.Node, chaosNS string, args ...string) error {
	for _, dropNode := range args {
		if node.IP == dropNode {
			// Don't drop itself
			continue
		}

		if err := n.t.Drop(ctx, node.IP, dropNode); err != nil {
			return err
		}
	}
	return nil
}

func (n drop) Recover(ctx context.Context, node cluster.Node, chaosNS string, args ...string) error {
	return n.t.Heal(ctx, node.IP)
}

func (drop) Name() string {
	return "drop"
}

func init() {
	core.RegisterNemesis(kill{})
	core.RegisterNemesis(drop{})
}

func createClient() (*Chaos, error) {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		return nil, err
	}
	return New(kubeCli), nil
}
