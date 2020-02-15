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
	return podKillNodes(nodes, len(nodes))
}

func (g podKillGenerator) Name() string {
	return string(core.PodKill)
}

func podKillNodes(nodes []cluster.Node, n int) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))
	// Note: 30s must pass, so we don't need to care about err.
	freq, _ := time.ParseDuration("30s")
	for i := 0; i < n; i++ {
		ops[indices[i]] = &core.NemesisOperation{
			Type:        core.PodKill,
			InvokeArgs:  []interface{}{nodes[i], freq},
			RecoverArgs: []interface{}{nodes[i]},
			// Note: Runtime means delay here.
			RunTime: time.Second * time.Duration(rand.Intn(120)+60),
		}
	}

	return ops
}

// NewPodKillGenerator creates a generator.
func NewPodKillGenerator(name string) core.NemesisGenerator {
	return podKillGenerator{name: name}
}

type podKill struct {
	k8sNemesisClient
}

func (k podKill) Invoke(ctx context.Context, _ cluster.Node, args ...interface{}) error {
	node, freq := extractPodKillArgs(args)
	log.Printf("Creating pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podKillTag(freq, node.Namespace, node.Namespace,
		node.PodName, chaosv1alpha1.PodKillAction)
	return k.cli.ApplyPodChaos(ctx, &podChaos)
}

func (k podKill) Recover(ctx context.Context, _ cluster.Node, args ...interface{}) error {
	node := extractKillArgs(args)
	log.Printf("Recover pod-kill with node %s(ns:%s)\n", node.PodName, node.Namespace)
	podChaos := podTag(node.Namespace, node.Namespace, node.PodName, chaosv1alpha1.PodKillAction)
	return k.cli.CancelPodChaos(ctx, &podChaos)
}

func (podKill) Name() string {
	return string(core.PodKill)
}

func extractPodKillArgs(args ...interface{}) (node cluster.Node, freq time.Time) {
	if args == nil || len(args) != 2 {
		panic("`extractPodKill` args is nil or not equal than two")
	}
	// validate arg1
	var ok bool
	if node, ok = args[0].(cluster.Node); !ok {
		panic("`extractPodKillArgs` received an typed error argument")
	}
	if freq, ok = args[1].(time.Time); !ok {
		panic("`extractPodKillArgs` received an typed error argument")
	}
	return
}

func podKillTag(freq time.Time, ns string, chaosNs string, name string, chaos chaosv1alpha1.PodChaosAction) chaosv1alpha1.PodChaos {
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
			Action:    chaos,
			Mode:      chaosv1alpha1.OnePodMode,
			Scheduler: &chaosv1alpha1.SchedulerSpec{Cron: "@every 30s"},
		},
	}
}
