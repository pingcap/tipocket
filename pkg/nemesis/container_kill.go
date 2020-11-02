package nemesis

import (
	"context"
	"math/rand"
	"strings"
	"time"

	chaosv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/ngaut/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type containerKillGenerator struct {
	name string
}

// Generate generates container-kill actions, to simulate the case that node can be recovered quickly after being killed
func (g containerKillGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	var n int
	var component *cluster.Component
	var freq = time.Second * time.Duration(rand.Intn(120)+60)
	switch g.name {
	case "short_kill_tikv_1node":
		n = 1
		cmp := cluster.TiKV
		component = &cmp
	case "short_kill_pd_leader":
		n = 1
		nodes = findPDMember(nodes, true)
		cmp := cluster.PD
		component = &cmp
	case "short_kill_tiflash_1node":
		n = 1
		cmp := cluster.TiFlash
		component = &cmp
	}
	return containerKillNodes(nodes, n, component, freq)
}

func (g containerKillGenerator) Name() string {
	return g.name
}

func containerKillNodes(nodes []cluster.Node, n int, component *cluster.Component, freq time.Duration) []*core.NemesisOperation {
	var ops []*core.NemesisOperation
	if component != nil {
		nodes = filterComponent(nodes, *component)
	}
	indices := shuffleIndices(len(nodes))
	if n > len(nodes) {
		n = len(nodes)
	}
	for i := 0; i < n; i++ {
		ops = append(ops, &core.NemesisOperation{
			Type:        core.ContainerKill,
			Node:        &nodes[indices[i]],
			InvokeArgs:  nil,
			RecoverArgs: nil,
			// Note: Container-Kill is a instant action, Runtime means the duration of next time container kill here.
			RunTime: freq,
		})
	}

	return ops
}

// NewContainerKillGenerator creates a generator.
func NewContainerKillGenerator(name string) core.NemesisGenerator {
	return containerKillGenerator{name: name}
}

// containerKill implements Nemesis
type containerKill struct {
	k8sNemesisClient
}

func (k containerKill) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Infof("apply nemesis container-kill on node %s(ns:%s)...", node.PodName, node.Namespace)
	containerChaos := buildContainerKillChaos(node.Namespace, node.Namespace,
		node.PodName, string(node.Component))
	return k.cli.ApplyPodChaos(ctx, &containerChaos)
}

func (k containerKill) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Infof("unapply container-kill on node %s(ns:%s)", node.PodName, node.Namespace)
	containerChaos := buildContainerKillChaos(node.Namespace, node.Namespace, node.PodName, string(node.Component))
	return k.cli.CancelPodChaos(ctx, &containerChaos)
}

func (containerKill) Name() string {
	return string(core.ContainerKill)
}

func buildContainerKillChaos(ns string, chaosNs string, podName string, containerName string) chaosv1alpha1.PodChaos {
	chaos := chaosv1alpha1.ContainerKillAction
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
			ContainerName: containerName,
			Action:        chaos,
			Mode:          chaosv1alpha1.OnePodMode,
			Scheduler:     &chaosv1alpha1.SchedulerSpec{Cron: "@every 1000000s"}, // let it be scheduled in our control
		},
	}
}
