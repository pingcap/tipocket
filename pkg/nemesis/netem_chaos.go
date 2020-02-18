package nemesis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type netemChaosGenerator struct {
	name string
}

func NewNetemChaos(name string) core.NemesisGenerator {
	return netemChaosGenerator{name: name}
}

// network loss
type loss struct{}

func (l loss) netemType() chaosv1alpha1.NetworkChaosAction {
	return chaosv1alpha1.LossAction
}

// The first arg means loss, the second args means correlation.
func (l loss) template(ns string, pods []string, podMode chaosv1alpha1.PodMode, args ...string) chaosv1alpha1.NetworkChaosSpec {
	if len(args) != 2 {
		panic("args number error")
	}
	return chaosv1alpha1.NetworkChaosSpec{
		Action: chaosv1alpha1.LossAction,
		Selector: chaosv1alpha1.SelectorSpec{
			Namespaces: []string{ns},
			Pods:       map[string][]string{ns: pods},
		},
		// default mode
		Mode: podMode,
		// default Loss
		Loss: &chaosv1alpha1.LossSpec{
			Loss:        args[0],
			Correlation: args[1],
		},
	}
}

func (l loss) defaultTemplate(ns string, pods []string) chaosv1alpha1.NetworkChaosSpec {
	return l.template(ns, pods, chaosv1alpha1.OnePodMode, "25", "25")
}

type delay struct{}

type netemChaos interface {
	netemType() chaosv1alpha1.NetworkChaosAction
	template(ns string, pods []string, podMode chaosv1alpha1.PodMode, args ...string) chaosv1alpha1.NetworkChaosSpec
	defaultTemplate(ns string, pods []string) chaosv1alpha1.NetworkChaosSpec
}

func selectNetem(name string) netemChaos {
	switch name {
	case "loss":
		return loss{}
	case "delay":
		panic("impl me")
	case "duplicate":
		panic("impl me")
	case "corrupt":
		panic("impl me")
	default:
		panic("selectNetem received an unexists tag")
	}
}

// Generate will randomly generate a chaos without selecting nodes.
func (g netemChaosGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	nChaos := selectNetem(g.name)
	ops := make([]*core.NemesisOperation, 1)

	ops[0] = &core.NemesisOperation{
		Type:        core.NetworkPartition,
		InvokeArgs:  []interface{}{nChaos},
		RecoverArgs: []interface{}{nChaos},
		RunTime:     time.Second * time.Duration(rand.Intn(120)+60),
	}

	return ops
}

func (g netemChaosGenerator) Name() string {
	return g.name
}

type netem struct {
	k8sNemesisClient
	chaos netemChaos
}

func (n netem) extractChaos(node cluster.Node, fnName string, args ...interface{}) chaosv1alpha1.NetworkChaos {
	log.Printf("%v was called", fnName)
	if len(args) != 1 {
		panic("netem.Invoke argument numbers of args is wrong")
	}
	var nChaos netemChaos
	var ok bool
	if nChaos, ok = args[0].(netemChaos); !ok {
		panic("netem.Invoke get wrong type")
	}
	networkChaosSpec := nChaos.defaultTemplate(node.Namespace, []string{node.PodName})
	return chaosv1alpha1.NetworkChaos{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", string(nChaos.netemType()), node.Namespace, node.PodName),
			Namespace: node.Namespace,
		},
		Spec: networkChaosSpec,
	}
}

func (n netem) Invoke(ctx context.Context, node cluster.Node, args ...interface{}) error {
	chaosSpec := n.extractChaos(node, "netem.Invoke", args...)
	return n.cli.ApplyNetChaos(&chaosSpec)
}

func (n netem) Recover(ctx context.Context, node cluster.Node, args ...interface{}) error {
	chaosSpec := n.extractChaos(node, "netem.Invoke", args...)
	return n.cli.CancelNetChaos(&chaosSpec)
}

func (n netem) Name() string {
	return string(n.chaos.netemType())
}
