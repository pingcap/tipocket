// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"fmt"

	chaosv1alpha1 "github.com/pingcap/chaos-mesh/api/v1alpha1"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/chaos"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var nemesisMap = make(map[string]NemesisFunc)

type NemesisFunc func(cli *chaos.Chaos, ns string, name string) error

func init() {
	nemesisMap["kill-pod"] = KillPod
	nemesisMap["network-delay"] = NetworkDelay
	nemesisMap["network-partition"] = NetworkPartition
}

func KillPod(cli *chaos.Chaos, ns string, name string) error {
	podkill := &chaosv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-kill-etcd",
			Namespace: ns,
		},
		Spec: chaosv1alpha1.PodChaosSpec{
			Selector: chaosv1alpha1.SelectorSpec{
				// Randomly kill in namespace
				Namespaces: []string{ns},
			},
			Scheduler: chaosv1alpha1.SchedulerSpec{
				Cron: "@every 30s",
			},
			Action: chaosv1alpha1.PodKillAction,
			Mode:   chaosv1alpha1.OnePodMode,
		},
	}

	return cli.ApplyPodChaos(podkill)
}

func NetworkDelay(cli *chaos.Chaos, ns string, name string) error {
	delay := &chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "network-delay-etcd",
			Namespace: ns,
		},
		Spec: chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.DelayAction,
			Mode:   chaosv1alpha1.FixedPodMode,
			Value:  "2",
			Selector: chaosv1alpha1.SelectorSpec{
				Namespaces: []string{ns},
			},
			Duration: "10s",
			Scheduler: chaosv1alpha1.SchedulerSpec{
				Cron: "@every 20s",
			},
			Delay: &chaosv1alpha1.DelaySpec{
				Latency:     "200ms",
				Correlation: "1",
				Jitter:      "10ms",
			},
		},
	}

	return cli.ApplyNetChaos(delay)
}

func NetworkPartition(cli *chaos.Chaos, ns string, name string) error {
	delay := &chaosv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "partition",
			Namespace: ns,
		},
		Spec: chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.PartitionAction,
			Mode:   chaosv1alpha1.AllPodMode,
			Selector: chaosv1alpha1.SelectorSpec{
				Pods: map[string][]string{
					ns: []string{
						fmt.Sprintf("%s-0", name),
						fmt.Sprintf("%s-1", name),
						fmt.Sprintf("%s-2", name),
					},
				},
			},
			Duration: "30s",
			Scheduler: chaosv1alpha1.SchedulerSpec{
				Cron: "@every 2m",
			},
			Direction: chaosv1alpha1.Both,
			Target: chaosv1alpha1.PartitionTarget{
				TargetSelector: chaosv1alpha1.SelectorSpec{
					Pods: map[string][]string{
						ns: []string{
							fmt.Sprintf("%s-3", name),
							fmt.Sprintf("%s-4", name),
						},
					},
				},
				TargetMode: chaosv1alpha1.AllPodMode,
			},
		},
	}

	return cli.ApplyNetChaos(delay)
}
