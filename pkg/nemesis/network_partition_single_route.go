package nemesis

import (
	"context"
	"fmt"
	"time"

	chaosv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/ngaut/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type networkPartitionSingleRouteGenerator struct {
	sourceNS  string
	sourcePod string
	targetNS  string
	targetPod string
	direction chaosv1alpha1.Direction
}

func NewNetworkPartitionSingleRoute(
	sourceNS string, sourcePod string, targetNS string, targetPod string, direction chaosv1alpha1.Direction,
) core.NemesisGenerator {
	return &networkPartitionSingleRouteGenerator{
		sourceNS:  sourceNS,
		sourcePod: sourcePod,
		targetNS:  targetNS,
		targetPod: targetPod,
		direction: direction,
	}
}

func (g *networkPartitionSingleRouteGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	name := fmt.Sprintf("%s-%s-%s", g.sourceNS, core.NetworkPartitionSingleRoute, randK8sObjectName())
	return []*core.NemesisOperation{{
		Type:        core.NetworkPartitionSingleRoute,
		InvokeArgs:  []interface{}{name, g.sourceNS, g.sourcePod, g.targetNS, g.targetPod, g.direction},
		RecoverArgs: []interface{}{name, g.sourceNS, g.sourcePod, g.targetNS, g.targetPod, g.direction},
		RunTime:     time.Hour * 10000, // Never recover before recovering manually
	}}
}

func (g *networkPartitionSingleRouteGenerator) Name() string {
	return "partition_route"
}

type networkPartitionSingleRoute struct {
	k8sNemesisClient
}

func (n networkPartitionSingleRoute) Invoke(ctx context.Context, _ *cluster.Node, args ...interface{}) error {
	name, sourceNS, sourcePod, targetNS, targetPod, direction := n.extractArgs(args...)
	log.Infof("apply nemesis %s %s between %v/%v and %v/%v (direction: %v)",
		core.NetworkPartitionSingleRoute, name, sourceNS, sourcePod, targetNS, targetPod, direction)
	return n.cli.ApplyNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sourceNS,
		},
		Spec: n.chaosSpec(sourceNS, sourcePod, targetNS, targetPod, direction),
	})
}

func (n networkPartitionSingleRoute) Recover(ctx context.Context, _ *cluster.Node, args ...interface{}) error {
	name, sourceNS, sourcePod, targetNS, targetPod, direction := n.extractArgs(args...)
	log.Infof("unapply nemesis %s %s between %v/%v and %v/%v (direction: %v)",
		core.NetworkPartitionSingleRoute, name, sourceNS, sourcePod, targetNS, targetPod, direction)
	return n.cli.CancelNetChaos(&chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sourceNS,
		},
		Spec: n.chaosSpec(sourceNS, sourcePod, targetNS, targetPod, direction),
	})
}

func (networkPartitionSingleRoute) Name() string {
	return string(core.NetworkPartitionSingleRoute)
}

func (networkPartitionSingleRoute) extractArgs(args ...interface{}) (name, sourceNS, sourcePod, targetNS, targetPod string, direction chaosv1alpha1.Direction) {
	if len(args) != 6 {
		log.Fatalf("wrong number of arguments for %v, got %+v", core.NetworkPartitionSingleRoute, args)
	}
	return args[0].(string), args[1].(string), args[2].(string), args[3].(string), args[4].(string), args[5].(chaosv1alpha1.Direction)
}

func (networkPartitionSingleRoute) chaosSpec(sourceNS, sourcePod, targetNS, targetPod string, direction chaosv1alpha1.Direction) chaosv1alpha1.NetworkChaosSpec {
	return chaosv1alpha1.NetworkChaosSpec{
		Action: chaosv1alpha1.PartitionAction,
		Mode:   chaosv1alpha1.AllPodMode,
		Selector: chaosv1alpha1.SelectorSpec{
			Pods: map[string][]string{
				sourceNS: {sourcePod},
			},
		},
		Direction: direction,
		Target: &chaosv1alpha1.Target{
			TargetSelector: chaosv1alpha1.SelectorSpec{
				Pods: map[string][]string{
					targetNS: {targetPod},
				},
			},
			TargetMode: chaosv1alpha1.AllPodMode,
		},
	}
}
