package nemesis

import (
	//"context"
	//"log"
	//"math/rand"
	//"strings"
	//"time"
	//
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	"math/rand"
	"time"

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
	}
}

func (g netemChaosGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1
	netem := selectNetem(g.name)
	ops := make([]*core.NemesisOperation, 1)

	ops[0] = &core.NemesisOperation{
		Type:        core.NetworkPartition,
		InvokeArgs:  []interface{}{netem},
		RecoverArgs: []interface{}{netem},
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

func (n netem) Name() string {
	return string(n.chaos.netemType())
}
