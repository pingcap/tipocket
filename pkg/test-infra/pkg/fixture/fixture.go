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

package fixture

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/scheme"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StorageType string

type E2eFixture struct {
	LocalVolumeStorageClass  string
	RemoteVolumeStorageClass string
	TiDBVersion              string
	MySQLVersion             string
	CDCImage                 string
	HubAddress               string
	DockerRepository         string
	ImageVersion             string
	BinlogVersion            string
	TimeLimit                time.Duration
	Nemesis                  string
	Workload                 string
	Round                    int
	RequestCount             int
	HistoryFile              string
}

var E2eContext E2eFixture

const (
	StorageTypeLocal  StorageType = "local"
	StorageTypeRemote StorageType = "remote"
	CPU                           = corev1.ResourceCPU
	Memory                        = corev1.ResourceMemory
	Storage                       = corev1.ResourceStorage
	EphemeralStorage              = corev1.ResourceEphemeralStorage
)

var (
	BestEffort = corev1.ResourceRequirements{}
	Small      = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
	}
	Medium = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("2000m"),
			Memory: resource.MustParse("4Gi"),
		},
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("2000m"),
			Memory: resource.MustParse("4Gi"),
		},
	}
	Large = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("4000m"),
			Memory: resource.MustParse("8Gi"),
		},
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("4000m"),
			Memory: resource.MustParse("8Gi"),
		},
	}
	XLarge = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("8000m"),
			Memory: resource.MustParse("16Gi"),
		},
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("8000m"),
			Memory: resource.MustParse("16Gi"),
		},
	}
)

func BuildGenericKubeClient(conf *rest.Config) (client.Client, error) {
	return client.New(conf, client.Options{
		Scheme: scheme.Scheme,
	})
}

func StorageClass(t StorageType) string {
	switch t {
	case StorageTypeLocal:
		return E2eContext.LocalVolumeStorageClass
	case StorageTypeRemote:
		return E2eContext.RemoteVolumeStorageClass
	default:
		return ""
	}
}

func WithStorage(r corev1.ResourceRequirements, size string) corev1.ResourceRequirements {
	if r.Requests == nil {
		r.Requests = corev1.ResourceList{}
	}

	r.Requests[Storage] = resource.MustParse(size)

	return r
}
