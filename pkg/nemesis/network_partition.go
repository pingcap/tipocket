package nemesis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type networkPartitionGenerator struct {
	name string
}

func (g networkPartitionGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1
	switch g.name {
	case "partition_one":
		n = 1
	default:
		n = 1
	}
	return partitionNodes(nodes, n)
}

func (g networkPartitionGenerator) Name() string {
	return g.name
}

func partitionNodes(nodes []cluster.Node, n int) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))

	var onePartNodes []cluster.Node
	var anotherPartNodes []cluster.Node

	for i := 0; i < n; i++ {
		onePartNodes = append(onePartNodes, nodes[indices[i]])
	}
	for i := n; i < len(nodes); i++ {
		anotherPartNodes = append(anotherPartNodes, nodes[indices[i]])
	}

	ops[0] = &core.NemesisOperation{
		Type:        core.NetworkPartition,
		InvokeArgs:  []interface{}{onePartNodes, anotherPartNodes},
		RecoverArgs: []interface{}{onePartNodes, anotherPartNodes},
		RunTime:     time.Second * time.Duration(rand.Intn(120)+60),
	}

	return ops
}

// NewNetworkPartitionGenerator creates a generator.
// Name is partition-one, etc.
func NewNetworkPartitionGenerator(name string) core.NemesisGenerator {
	return networkPartitionGenerator{name: name}
}

type networkPartition struct {
	k8sNemesisClient
}

func (n networkPartition) Invoke(ctx context.Context, _ cluster.Node, args ...interface{}) error {
	onePart, anotherPart := extractTwoParts(args...)
	return n.cli.ApplyNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", onePart[0].Namespace, chaosv1alpha1.PartitionAction),
			Namespace: onePart[0].Namespace,
		},
		Spec: chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.PartitionAction,
			Mode:   chaosv1alpha1.AllPodMode,
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: map[string][]string{
					onePart[0].Namespace: extractPodNames(onePart),
				},
			},
			Direction: chaosv1alpha1.Both,
			Target: chaosv1alpha1.PartitionTarget{
				TargetSelector: chaosv1alpha1.SelectorSpec{
					Pods: map[string][]string{
						anotherPart[0].Namespace: extractPodNames(anotherPart),
					},
				},
				TargetMode: chaosv1alpha1.AllPodMode,
			},
		},
	})
}

func (n networkPartition) Recover(ctx context.Context, _ cluster.Node, args ...interface{}) error {
	onePart, anotherPart := extractTwoParts(args...)
	return n.cli.CancelNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", onePart[0].Namespace, chaosv1alpha1.PartitionAction),
			Namespace: onePart[0].Namespace,
		},
		Spec: chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.PartitionAction,
			Mode:   chaosv1alpha1.AllPodMode,
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: map[string][]string{
					onePart[0].Namespace: extractPodNames(onePart),
				},
			},
			Direction: chaosv1alpha1.Both,
			Target: chaosv1alpha1.PartitionTarget{
				TargetSelector: chaosv1alpha1.SelectorSpec{
					Pods: map[string][]string{
						anotherPart[0].Namespace: extractPodNames(anotherPart),
					},
				},
				TargetMode: chaosv1alpha1.AllPodMode,
			},
		},
	})
}

func (n networkPartition) Name() string {
	return string(core.NetworkPartition)
}

func extractTwoParts(args ...interface{}) ([]cluster.Node, []cluster.Node) {
	var networkParts [][]cluster.Node
	var onePart []cluster.Node
	var anotherPart []cluster.Node

	for _, arg := range args {
		networkPart := arg.([]cluster.Node)
		networkParts = append(networkParts, networkPart)
	}

	if len(networkParts) != 2 {
		log.Panicf("expect two network parts, got %+v", networkParts)
	}
	onePart = networkParts[0]
	anotherPart = networkParts[1]
	if len(onePart) < 1 || len(anotherPart) < 1 {
		log.Panicf("expect non-empty two parts, got %+v and %+v", onePart, anotherPart)
	}
	return onePart, anotherPart
}

func extractPodNames(nodes []cluster.Node) []string {
	var podNames []string

	for _, node := range nodes {
		podNames = append(podNames, node.PodName)
	}
	return podNames
}
