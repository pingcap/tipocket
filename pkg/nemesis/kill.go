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
	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

// killGenerator generate PodFailure chaos.
type killGenerator struct {
	name string
}

// Generate generates container-kill actions, to simulate the case that node can't be recovered quickly after being killed
func (g killGenerator) Generate(nodes []clusterTypes.Node) []*core.NemesisOperation {
	var n int
	var duration = time.Second * time.Duration(rand.Intn(120)+60)
	var component *clusterTypes.Component

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
	case "kill_pd_leader_5min":
		n = 1
		duration = time.Minute * time.Duration(5)
		cmp := cluster.PD
		component = &cmp
		nodes = findPDMember(nodes, true)
	case "kill_pd_non_leader_5min":
		n = 1
		duration = time.Minute * time.Duration(5)
		cmp := cluster.PD
		component = &cmp
		nodes = findPDMember(nodes, false)
	default:
		n = 1
	}
	return killNodes(nodes, n, component, duration)
}

func (g killGenerator) Name() string {
	return g.name
}

func killNodes(nodes []clusterTypes.Node, n int, component *cluster.Component, duration time.Duration) []*core.NemesisOperation {
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

func findPDMember(nodes []cluster.Node, ifLeader bool) []cluster.Node {
	var (
		leader string
		err    error
		result []cluster.Node
	)
	for _, node := range nodes {
		if node.Component == cluster.PD {
			if leader == "" && node.Client != nil {
				leader, _, err = node.PDMember()
				if err != nil {
					log.Fatalf("find pd members occured an error: %+v", err)
				}
			}
			if ifLeader && node.PodName == leader {
				return []cluster.Node{node}
			}
			if !ifLeader && node.PodName != leader {
				result = append(result, node)
			}
		}
	}

	return result
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

func (k kill) Invoke(ctx context.Context, node *clusterTypes.Node, _ ...interface{}) error {
	log.Printf("Creating pod-failure with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := buildPodFailureChaos(node.Namespace, node.Namespace, node.PodName)
	return k.cli.ApplyPodChaos(ctx, &podChaos)
}

func (k kill) Recover(ctx context.Context, node *clusterTypes.Node, _ ...interface{}) error {
	log.Printf("Recover pod-failure with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := buildPodFailureChaos(node.Namespace, node.Namespace, node.PodName)
	return k.cli.CancelPodChaos(ctx, &podChaos)
}

func (kill) Name() string {
	return string(core.PodFailure)
}

func buildPodFailureChaos(ns string, chaosNs string, name string) chaosv1alpha1.PodChaos {
	var chaos = chaosv1alpha1.PodFailureAction
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
