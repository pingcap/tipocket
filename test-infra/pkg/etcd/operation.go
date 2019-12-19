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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ETCDOps struct {
	cli client.Client
}

func New(cli client.Client) *ETCDOps {
	return &ETCDOps{cli}
}

type ETCDSpec struct {
	Name      string
	Namespace string
	Version   string
	Replicas  int
}

type ETCD struct {
	Sts *appsv1.StatefulSet
	Svc *corev1.Service
}

func (e *ETCDOps) ApplyETCD(spec *ETCDSpec) (*ETCD, error) {
	return nil, nil
}

func (e *ETCDOps) renderETCD() (*ETCD, error) {
	return nil, nil
}
