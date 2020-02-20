package nemesis

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

// killGenerator generate code about PodFailure chaos.
type killGenerator struct {
	name string
}

func (g killGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	var n = 1
	var duration = time.Second * time.Duration(rand.Intn(120)+60)
	var component *cluster.Component

	// This part decide how many machines to apply pod-failure
	switch g.name {
	case "minor_kill":
		n = len(nodes)/2 - 1
	case "major_kill":
		n = len(nodes)/2 + 1
	case "all_kill":
		n = len(nodes)
	case "kill_tikv_1node_5min":
		n = 1
		duration = time.Minute * time.Duration(5)
		cmp := cluster.TiKV
		component = &cmp
	case "kill_tikv_2node_5min":
		n = 2
		duration = time.Minute * time.Duration(5)
		cmp := cluster.TiKV
		component = &cmp
	default:
		n = 1
	}
	return killNodes(nodes, n, component, duration)
}

func (g killGenerator) Name() string {
	return g.name
}

func killNodes(nodes []cluster.Node, n int, component *cluster.Component, duration time.Duration) []*core.NemesisOperation {
	var ops []*core.NemesisOperation
	if component != nil {
		nodes = filterComponent(nodes, *component)
	}
	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))
	if n > len(indices) {
		n = len(indices)
	}
	for i := 0; i < n; i++ {
		ops = append(ops, &core.NemesisOperation{
			Type:        core.PodFailure,
			Node:        &nodes[indices[i]],
			InvokeArgs:  nil,
			RecoverArgs: nil,
			RunTime:     duration,
		})
	}

	return ops
}

func filterComponent(nodes []cluster.Node, component cluster.Component) []cluster.Node {
	var componentNodes []cluster.Node

	for _, node := range nodes {
		if node.Component == component {
			componentNodes = append(componentNodes, node)
		}
	}

	return componentNodes
}

// NewKillGenerator creates a generator.
// Name is random_kill, minor_kill, major_kill, and all_kill.
func NewKillGenerator(name string) core.NemesisGenerator {
	return killGenerator{name: name}
}

// kill implements Nemesis
type kill struct {
	k8sNemesisClient
}

func (k kill) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("Creating pod-failure with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podTag(node.Namespace, node.Namespace, node.PodName, chaosv1alpha1.PodFailureAction)
	return k.cli.ApplyPodChaos(ctx, &podChaos)
}

func (k kill) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Printf("Recover pod-failure with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podTag(node.Namespace, node.Namespace, node.PodName, chaosv1alpha1.PodFailureAction)
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
			Name:      strings.Join([]string{name, string(chaos)}, "-"),
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
