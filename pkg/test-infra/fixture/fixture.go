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
	"flag"
	"net/http"
	_ "net/http/pprof" // pprof
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/test-infra/scheme"
)

type StorageType string

type fixtureContext struct {
	// Control config
	Mode         int
	ClientCount  int
	Nemesis      string
	RunRound     int
	RunTime      time.Duration
	RequestCount int
	HistoryFile  string
	// Test-infra
	Namespace                string
	WaitClusterReadyDuration time.Duration
	Purge                    bool
	ABTestConfig             ABTestConfig
	BinlogConfig             BinlogConfig
	LocalVolumeStorageClass  string
	TiDBMonitorSvcType       string
	RemoteVolumeStorageClass string
	TiDBVersion              string
	MySQLVersion             string
	CDCImage                 string
	HubAddress               string
	DockerRepository         string
	ImageVersion             string
	TiDBConfigFile           string
	TiKVConfigFile           string
	PDConfigFile             string
	// Loki
	LokiAddress  string
	LokiUsername string
	LokiPassword string
	// Other
	pprofAddr string
}

var Context fixtureContext

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
		return Context.LocalVolumeStorageClass
	case StorageTypeRemote:
		return Context.RemoteVolumeStorageClass
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

func init() {
	flag.IntVar(&Context.Mode, "mode", 0, "control mode, 0: mixed, 1: sequential mode, 2: self scheduled mode")
	flag.IntVar(&Context.ClientCount, "client", 5, "client count")
	flag.StringVar(&Context.Nemesis, "nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	flag.IntVar(&Context.RunRound, "round", 1, "run round of client test")
	flag.DurationVar(&Context.RunTime, "run-time", 100*time.Minute, "run time of client")
	flag.IntVar(&Context.RequestCount, "request-count", 10000, "requests a client sends to the db")
	flag.StringVar(&Context.HistoryFile, "history", "./history.log", "history file record client operation")

	flag.StringVar(&Context.Namespace, "namespace", "", "test namespace")
	flag.StringVar(&Context.HubAddress, "hub", "", "hub address, default to docker hub")
	flag.StringVar(&Context.LocalVolumeStorageClass, "storage-class", "local-storage", "storage class name")
	flag.StringVar(&Context.TiDBMonitorSvcType, "monitor-svc", "ClusterIP", "TiDB monitor service type")
	flag.StringVar(&Context.pprofAddr, "pprof", "0.0.0.0:8080", "Pprof address")
	flag.StringVar(&Context.BinlogConfig.BinlogVersion, "binlog-version", "", `overwrite "-image-version" flag for drainer`)
	flag.BoolVar(&Context.BinlogConfig.EnableRelayLog, "relay-log", false, "if enable relay log")
	flag.DurationVar(&Context.WaitClusterReadyDuration, "wait-duration", 4*time.Hour, "clusters ready wait duration")
	flag.BoolVar(&Context.Purge, "purge", false, "purge the whole cluster on success")

	flag.StringVar(&Context.LokiAddress, "loki-addr", "", "loki address. If empty then don't query logs from loki.")
	flag.StringVar(&Context.LokiUsername, "loki-username", "", "loki username. Needed when basic auth is configured in loki")
	flag.StringVar(&Context.LokiPassword, "loki-password", "", "loki password. Needed when basic auth is configured in loki")

	flag.StringVar(&Context.ImageVersion, "image-version", "latest", "image version")
	flag.StringVar(&Context.TiDBConfigFile, "tidb-configmap", "", "path of tidb configmap (cluster A in abtest case)")
	flag.StringVar(&Context.TiKVConfigFile, "tikv-configmap", "", "path of tikv configmap (cluster A in abtest case)")
	flag.StringVar(&Context.PDConfigFile, "pd-configmap", "", "path of pd configmap (cluster A in abtest case)")
	flag.StringVar(&Context.ABTestConfig.TiDBConfigFile, "abtest.b.tidb-configmap", "", "tidb configmap for cluster b")
	flag.StringVar(&Context.ABTestConfig.TiKVConfigFile, "abtest.b.tikv-configmap", "", "tikv configmap for cluster b")
	flag.StringVar(&Context.ABTestConfig.PDConfigFile, "abtest.b.pd-configmap", "", "pd configmap for cluster b")
	flag.StringVar(&Context.ABTestConfig.Cluster1Version, "abtest.a.version", "", "specify version for cluster a")
	flag.StringVar(&Context.ABTestConfig.Cluster2Version, "abtest.b.version", "", "specify version for cluster b")
	flag.StringVar(&Context.ABTestConfig.LogPath, "abtest.log", "", "log path for abtest, default to stdout")

	Context.DockerRepository = "pingcap"

	go func() {
		http.ListenAndServe(Context.pprofAddr, nil)
	}()

}
