package nemesis

import (
	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PodTag(ns string, name string, chaos chaosv1alpha1.PodChaosAction) chaosv1alpha1.PodChaos {
	return chaosv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: chaosv1alpha1.PodChaosSpec{
			Selector: chaosv1alpha1.SelectorSpec{
				// Randomly kill in namespace
				Namespaces: []string{ns},
			},
			// TODO: using args to adapt this
			Scheduler: &chaosv1alpha1.SchedulerSpec{
				Cron: "@every 30s",
			},
			Action: chaos,
			Mode:   chaosv1alpha1.OnePodMode,
		},
	}
}

func PodChaos(cli *Chaos, ns string, name string, chaos chaosv1alpha1.PodChaosAction) error {
	podchaos := PodTag(ns, name, chaos)

	return cli.ApplyPodChaos(&podchaos)
}

func CancelPodChaos(cli *Chaos, ns string, name string, chaos chaosv1alpha1.PodChaosAction) error {
	podchaos := PodTag(ns, name, chaos)

	return cli.CancelPodChaos(&podchaos)
}

//func NetworkDelay(cli *Chaos, ns string, name string) error {
//	delay := &chaosv1alpha1.NetworkChaos{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "network-delay-etcd",
//			Namespace: ns,
//		},
//		Spec: chaosv1alpha1.NetworkChaosSpec{
//			Action: chaosv1alpha1.DelayAction,
//			Mode:   chaosv1alpha1.FixedPodMode,
//			Value:  "2",
//			Selector: chaosv1alpha1.SelectorSpec{
//				Namespaces: []string{ns},
//			},
//			Duration: "10s",
//			Scheduler: chaosv1alpha1.SchedulerSpec{
//				Cron: "@every 20s",
//			},
//			Delay: &chaosv1alpha1.DelaySpec{
//				Latency:     "200ms",
//				Correlation: "1",
//				Jitter:      "10ms",
//			},
//		},
//	}
//
//	return cli.ApplyNetChaos(delay)
//}
//
//func NetworkPartition(cli *Chaos, ns string, name string) error {
//	delay := &chaosv1alpha1.NetworkChaos{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "partition",
//			Namespace: ns,
//		},
//		Spec: chaosv1alpha1.NetworkChaosSpec{
//			Action: chaosv1alpha1.PartitionAction,
//			Mode:   chaosv1alpha1.AllPodMode,
//			Selector: chaosv1alpha1.SelectorSpec{
//				Pods: map[string][]string{
//					ns: []string{
//						fmt.Sprintf("%s-0", name),
//						fmt.Sprintf("%s-1", name),
//						fmt.Sprintf("%s-2", name),
//					},
//				},
//			},
//			Duration: "30s",
//			Scheduler: chaosv1alpha1.SchedulerSpec{
//				Cron: "@every 2m",
//			},
//			Direction: chaosv1alpha1.Both,
//			Target: chaosv1alpha1.PartitionTarget{
//				TargetSelector: chaosv1alpha1.SelectorSpec{
//					Pods: map[string][]string{
//						ns: []string{
//							fmt.Sprintf("%s-3", name),
//							fmt.Sprintf("%s-4", name),
//						},
//					},
//				},
//				TargetMode: chaosv1alpha1.AllPodMode,
//			},
//		},
//	}
//
//	return cli.ApplyNetChaos(delay)
//}
