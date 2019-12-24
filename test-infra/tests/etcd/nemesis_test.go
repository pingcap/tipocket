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
	chaosv1alpha1 "github.com/pingcap/chaos-operator/api/v1alpha1"

	"github.com/pingcap/tipocket/test-infra/pkg/chaos"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var nemesisMap = make(map[string]NemesisFunc)

type NemesisFunc func(cli *chaos.Chaos, ns string) error

func init() {
	nemesisMap["kill-pod"] = KillPod
}

func KillPod(cli *chaos.Chaos, ns string) error {
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

func networkDelay(cli *chaos.Chaos, ns string) error {
	delay := &chaosv1alpha1.NetworkChaos{

	}
}
