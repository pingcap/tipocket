package nemesis

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/chaos-mesh/api/v1alpha1"
	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type killGenerator struct {
	name string
}

func (g killGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1

	// This part decide how many machines to apply pod-failure
	switch g.name {
	case "minor_kill":
		n = len(nodes)/2 - 1
	case "major_kill":
		n = len(nodes)/2 + 1
	case "all_kill":
		n = len(nodes)
	default:
		n = 1
	}

	return killNodes(nodes, n)
}

func (g killGenerator) Name() string {
	return string(core.PodFailure)
}

func killNodes(nodes []cluster.Node, n int) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))

	for i := 0; i < n; i++ {
		ops[indices[i]] = &core.NemesisOperation{
			Type: core.PodFailure,
			// Note: Maybe I should just store cluster info here.
			InvokeArgs:  nil,
			RecoverArgs: nil,
			RunTime:     time.Second * time.Duration(rand.Intn(120)+60),
		}
	}

	return ops
}

// NewKillGenerator creates a generator.
// Name is random_kill, minor_kill, major_kill, and all_kill.
func NewKillGenerator(name string) core.NemesisGenerator {
	return killGenerator{name: name}
}

type kill struct {
	k8sNemesisClient
}

func (k kill) Invoke(ctx context.Context, node cluster.Node, args ...interface{}) error {
	log.Printf("Creating pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podTag(node.Namespace, node.Namespace, node.PodName, v1alpha1.PodFailureAction)
	return k.cli.ApplyPodChaos(ctx, &podChaos)
}

func (k kill) Recover(ctx context.Context, node cluster.Node, args ...interface{}) error {
	log.Printf("Recover pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podTag(node.Namespace, node.Namespace, node.PodName, v1alpha1.PodFailureAction)
	return k.cli.CancelPodChaos(ctx, &podChaos)
}

func (kill) Name() string {
	return string(core.PodFailure)
}

func podTag(ns string, chaosNs string, name string, chaos chaosv1alpha1.PodChaosAction) chaosv1alpha1.PodChaos {
	pods := make(map[string][]string)
	pods[ns] = []string{name}

	return chaosv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Join([]string{name, string(chaos)}, "-"),
			// TODO: this might be a chaos ns, so let's take it carefully.
			Namespace: chaosNs,
		},
		Spec: chaosv1alpha1.PodChaosSpec{
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: pods,
			},
			Action: chaos,
			Mode:   chaosv1alpha1.OnePodMode,
		},
	}
}
