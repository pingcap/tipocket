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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tipocket/test-infra/pkg/scheme"

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
	DockerRepository         string
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
)

var (
	BestEffort = v1alpha1.Resources{}
	Small      = v1alpha1.Resources{
		Requests: &v1alpha1.ResourceRequirement{
			CPU:    "1000m",
			Memory: "1Gi",
		},
		Limits: &v1alpha1.ResourceRequirement{
			CPU:    "1000m",
			Memory: "1Gi",
		},
	}
	Medium = v1alpha1.Resources{
		Requests: &v1alpha1.ResourceRequirement{
			CPU:    "2000m",
			Memory: "4Gi",
		},
		Limits: &v1alpha1.ResourceRequirement{
			CPU:    "2000m",
			Memory: "4Gi",
		},
	}
	Large = v1alpha1.Resources{
		Requests: &v1alpha1.ResourceRequirement{
			CPU:    "4000m",
			Memory: "8Gi",
		},
		Limits: &v1alpha1.ResourceRequirement{
			CPU:    "4000m",
			Memory: "8Gi",
		},
	}
	XLarge = v1alpha1.Resources{
		Requests: &v1alpha1.ResourceRequirement{
			CPU:    "8000m",
			Memory: "16Gi",
		},
		Limits: &v1alpha1.ResourceRequirement{
			CPU:    "8000m",
			Memory: "16Gi",
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

func WithStorage(r v1alpha1.Resources, size string) v1alpha1.Resources {
	if r.Requests == nil {
		r.Requests = &v1alpha1.ResourceRequirement{}
	}
	r.Requests.Storage = size

	return r
}
