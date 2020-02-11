package nemesis

import (
	"context"

	"github.com/pingcap/chaos-mesh/api/v1alpha1"
	"k8s.io/client-go/rest"

	"github.com/pingcap/tipocket/pkg/nemesis/scheme"

	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/net"
)

type kill struct{}

func (kill) Invoke(ctx context.Context, node cluster.Node, args ...string) error {
	c := createClient(node.Namespace)
	return PodChaos(ctx, c, node.Namespace, node.PodName, v1alpha1.PodKillAction)
}

func (kill) Recover(ctx context.Context, node cluster.Node, args ...string) error {
	c := createClient(node.Namespace)
	return CancelPodChaos(ctx, c, node.Namespace, node.PodName, v1alpha1.PodKillAction)
}

func (kill) Name() string {
	return "kill"
}

type drop struct {
	t net.IPTables
}

func (n drop) Invoke(ctx context.Context, node cluster.Node, args ...string) error {
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

func (n drop) Recover(ctx context.Context, node cluster.Node, args ...string) error {
	return n.t.Heal(ctx, node.IP)
}

func (drop) Name() string {
	return "drop"
}

func init() {
	core.RegisterNemesis(kill{})
	core.RegisterNemesis(drop{})
}

func newClient(conf *rest.Config) *Chaos {
	kubeCli, err := client.New(conf, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		e2elog.Failf("error creating kube-client: %v", err)
	}
	return &Chaos{
		cli: kubeCli,
	}
}

func createClient(ns string) *Chaos {
	e2elog.Logf("Working namespace %s", ns)
	var err error
	conf, err := framework.LoadConfig()
	framework.ExpectNoError(err, "Expected to load config.")
	c := newClient(conf)
	return c
}
