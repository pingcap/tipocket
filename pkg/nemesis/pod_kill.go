package nemesis

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

// podKillGenerator generate code about PodKill chaos.
type podKillGenerator struct {
	name string
}

func (g podKillGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	return podKillNodes(nodes, len(nodes), time.Second*time.Duration(rand.Intn(120)+60))
}

func (g podKillGenerator) Name() string {
	return string(core.PodKill)
}

func podKillNodes(nodes []cluster.Node, n int, freq time.Duration) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))
	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))
	for i := 0; i < n; i++ {
		ops[indices[i]] = &core.NemesisOperation{
			Type:        core.PodKill,
			Node:        &nodes[indices[i]],
			InvokeArgs:  nil,
			RecoverArgs: nil,
			// Note: PodKill is a instant action, Runtime means the duration of next time pod kill here.
			RunTime: freq,
		}
	}

	return ops
}

// NewPodKillGenerator creates a generator.
func NewPodKillGenerator(name string) core.NemesisGenerator {
	return podKillGenerator{name: name}
}

// podKill implements Nemesis
type podKill struct {
	k8sNemesisClient
}

func (k podKill) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("Creating pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := buildPodKillChaos(node.Namespace, node.Namespace,
		node.PodName)
	return k.cli.ApplyPodChaos(ctx, &podChaos)
}

func (k podKill) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("Recover pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := buildPodKillChaos(node.Namespace, node.Namespace, node.PodName)
	return k.cli.CancelPodChaos(ctx, &podChaos)
}

func (podKill) Name() string {
	return string(core.PodKill)
}

func buildPodKillChaos(ns string, chaosNs string, podName string) chaosv1alpha1.PodChaos {
	chaos := chaosv1alpha1.PodKillAction
	pods := make(map[string][]string)
	pods[ns] = []string{podName}
	return chaosv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{podName, string(chaos)}, "-"),
			Namespace: chaosNs,
		},
		Spec: chaosv1alpha1.PodChaosSpec{
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: pods,
			},
			Action:    chaos,
			Mode:      chaosv1alpha1.OnePodMode,
			Scheduler: &chaosv1alpha1.SchedulerSpec{Cron: "@every 1000000s"}, // let it be scheduled in our control
		},
	}
}
