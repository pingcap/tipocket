package nemesis

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ngaut/log"
	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

type networkPartitionGenerator struct {
	name string
}

func (g networkPartitionGenerator) Generate(nodes []clusterTypes.Node) []*core.NemesisOperation {
	n := 1
	switch g.name {
	case "partition_one":
		n = 1
	default:
		n = 1
	}
	return partitionNodes(nodes, n, time.Second*time.Duration(rand.Intn(120)+60))
}

func (g networkPartitionGenerator) Name() string {
	return g.name
}

func partitionNodes(nodes []clusterTypes.Node, n int, duration time.Duration) []*core.NemesisOperation {
	if n < 1 {
		log.Fatalf("the partition part size cannot be less than 1")
	}
	var ops []*core.NemesisOperation
	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))

	var onePartNodes []clusterTypes.Node
	var anotherPartNodes []clusterTypes.Node
	for i := 0; i < n; i++ {
		onePartNodes = append(onePartNodes, nodes[indices[i]])
	}
	for i := n; i < len(nodes); i++ {
		anotherPartNodes = append(anotherPartNodes, nodes[indices[i]])
	}

	name := fmt.Sprintf("%s-%s-%s", onePartNodes[0].Namespace, core.NetworkPartition, randK8sObjectName())
	ops = append(ops, &core.NemesisOperation{
		Type:        core.NetworkPartition,
		InvokeArgs:  []interface{}{name, onePartNodes, anotherPartNodes},
		RecoverArgs: []interface{}{name, onePartNodes, anotherPartNodes},
		RunTime:     duration,
	})

	return ops
}

// NewNetworkPartitionGenerator creates a generator.
// Name is partition-one, etc.
func NewNetworkPartitionGenerator(name string) core.NemesisGenerator {
	return networkPartitionGenerator{name: name}
}

// networkPartition implements Nemesis
type networkPartition struct {
	k8sNemesisClient
}

func networkChaosSpecTemplate(partOneNs, partTwoNS string, partOne, partTwo []clusterTypes.Node) chaosv1alpha1.NetworkChaosSpec {
	return chaosv1alpha1.NetworkChaosSpec{
		Action: chaosv1alpha1.PartitionAction,
		Mode:   chaosv1alpha1.AllPodMode,
		Selector: chaosv1alpha1.SelectorSpec{
			Pods: map[string][]string{
				partOneNs: extractPodNames(partOne),
			},
		},
		Direction: chaosv1alpha1.Both,
		Target: chaosv1alpha1.PartitionTarget{
			TargetSelector: chaosv1alpha1.SelectorSpec{
				Pods: map[string][]string{
					partTwoNS: extractPodNames(partTwo),
				},
			},
			TargetMode: chaosv1alpha1.AllPodMode,
		},
	}
}

func (n networkPartition) Invoke(ctx context.Context, _ *clusterTypes.Node, args ...interface{}) error {
	name, onePart, anotherPart := extractArgs(args...)
	log.Infof("apply nemesis %s %s between %+v and %+v", core.NetworkPartition, name, onePart, anotherPart)
	return n.cli.ApplyNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: onePart[0].Namespace,
		},
		Spec: networkChaosSpecTemplate(onePart[0].Namespace,
			anotherPart[0].Namespace, onePart, anotherPart),
	})
}

func (n networkPartition) Recover(ctx context.Context, _ *clusterTypes.Node, args ...interface{}) error {
	name, onePart, anotherPart := extractArgs(args...)
	log.Infof("unapply nemesis %s %s between %+v and %+v", core.NetworkPartition, name, onePart, anotherPart)
	return n.cli.CancelNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: onePart[0].Namespace,
		},
		Spec: networkChaosSpecTemplate(onePart[0].Namespace,
			anotherPart[0].Namespace, onePart, anotherPart),
	})
}

func (n networkPartition) Name() string {
	return string(core.NetworkPartition)
}

func extractArgs(args ...interface{}) (string, []clusterTypes.Node, []clusterTypes.Node) {
	var name = args[0].(string)
	var networkParts [][]clusterTypes.Node
	var onePart []clusterTypes.Node
	var anotherPart []clusterTypes.Node

	for _, arg := range args[1:] {
		networkPart := arg.([]clusterTypes.Node)
		networkParts = append(networkParts, networkPart)
	}

	if len(networkParts) != 2 {
		log.Fatalf("expect two network parts, got %+v", networkParts)
	}
	onePart = networkParts[0]
	anotherPart = networkParts[1]
	if len(onePart) < 1 || len(anotherPart) < 1 {
		log.Fatalf("expect non-empty two parts, got %+v and %+v", onePart, anotherPart)
	}
	return name, onePart, anotherPart
}

func extractPodNames(nodes []clusterTypes.Node) []string {
	var podNames []string

	for _, node := range nodes {
		podNames = append(podNames, node.PodName)
	}
	return podNames
}
