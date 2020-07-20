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
	"strings"
	"time"

	"github.com/ngaut/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/test-infra/scheme"
)

var (
	// BuildTS ...
	BuildTS = "None"
	// BuildHash ...
	BuildHash = "None"
)

// StorageType ...
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
	DeleteNS                 bool
	LocalVolumeStorageClass  string
	TiDBMonitorSvcType       string
	RemoteVolumeStorageClass string
	MySQLVersion             string
	HubAddress               string
	DockerRepository         string
	TiDBClusterConfig        TiDBClusterConfig
	BinlogConfig             BinlogConfig
	CDCConfig                CDCConfig
	DMConfig                 DMConfig
	TiFlashConfig            TiFlashConfig
	ABTestConfig             ABTestConfig
	// Loki
	LokiAddress  string
	LokiUsername string
	LokiPassword string
	// Other
	pprofAddr  string
	EnableHint bool
	LogPath    string
	// Plugins
	LeakCheckEatFile string
	LeakCheckSilent  bool
	// failpoints
	TiDBFailpoint string
}

type addressArrayFlags []string

func (a *addressArrayFlags) String() string {
	return "multiple addresses"
}

func (a *addressArrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

type fileArrayFlags []string

func (a *fileArrayFlags) String() string {
	return "multiple files"
}

func (a *fileArrayFlags) Set(value string) error {
	for _, value := range strings.Split(value, ";") {
		*a = append(*a, value)
	}
	return nil
}

// TiDBClusterConfig ...
type TiDBClusterConfig struct {
	// hub address
	TiDBHubAddress    string
	TiKVHubAddress    string
	PDHubAddress      string
	TiFlashHubAddress string

	// image versions
	ImageVersion string
	TiDBImage    string
	TiKVImage    string
	PDImage      string
	TiFlashImage string

	// configurations
	TiDBConfig string
	TiKVConfig string
	PDConfig   string

	// replicas
	TiDBReplicas    int
	TiKVReplicas    int
	PDReplicas      int
	TiFlashReplicas int

	// Database address
	TiDBAddr addressArrayFlags
	TiKVAddr addressArrayFlags
	PDAddr   addressArrayFlags

	MatrixConfig MatrixConfig
}

// Context ...
var Context fixtureContext

const (
	// StorageTypeLocal ...
	StorageTypeLocal StorageType = "local"
	// StorageTypeRemote ...
	StorageTypeRemote StorageType = "remote"
	// CPU ...
	CPU = corev1.ResourceCPU
	// Memory ...
	Memory = corev1.ResourceMemory
	// Storage ...
	Storage = corev1.ResourceStorage
)

var (
	// BestEffort ...
	BestEffort = corev1.ResourceRequirements{}
	// Small ...
	Small = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("1000m"),
			Memory: resource.MustParse("1Gi"),
		},
	}
	// Medium ...
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
	// Large ...
	Large = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			CPU:    resource.MustParse("4000m"),
			Memory: resource.MustParse("4Gi"),
		},
		Limits: corev1.ResourceList{
			CPU:    resource.MustParse("4000m"),
			Memory: resource.MustParse("16Gi"),
		},
	}
	// XLarge ...
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

// BuildGenericKubeClient builds kube client
func BuildGenericKubeClient(conf *rest.Config) (client.Client, error) {
	return client.New(conf, client.Options{
		Scheme: scheme.Scheme,
	})
}

// StorageClass ...
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

// WithStorage ...
func WithStorage(r corev1.ResourceRequirements, size string) corev1.ResourceRequirements {
	if r.Requests == nil {
		r.Requests = corev1.ResourceList{}
	}

	r.Requests[Storage] = resource.MustParse(size)

	return r
}

func printVersion() {
	log.Info("Git Commit Hash:", BuildHash)
	log.Info("UTC Build Time:", BuildTS)
}

func init() {
	printVersion()

	flag.IntVar(&Context.Mode, "mode", 0, "control mode, 0: mixed, 1: sequential mode, 2: self scheduled mode")
	flag.IntVar(&Context.ClientCount, "client", 5, "client count")
	// (TODO:yeya24) Now nemesis option is only for one TiDBCluster. If we want to add nemesis in AB Test,
	// we can add another option for ClusterB.
	flag.StringVar(&Context.Nemesis, "nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	flag.IntVar(&Context.RunRound, "round", 1, "run round of client test")
	flag.DurationVar(&Context.RunTime, "run-time", 100*time.Minute, "run time of client")
	flag.IntVar(&Context.RequestCount, "request-count", 10000, "requests a client sends to the db")
	flag.StringVar(&Context.HistoryFile, "history", "./history.log", "history file record client operation")

	flag.StringVar(&Context.Namespace, "namespace", "", "test namespace")
	flag.StringVar(&Context.MySQLVersion, "mysql-version", "5.6", "Default mysql version")
	flag.StringVar(&Context.DockerRepository, "repository", "pingcap", "repo name, default is pingcap")
	flag.StringVar(&Context.LocalVolumeStorageClass, "storage-class", "local-storage", "storage class name")
	flag.StringVar(&Context.TiDBMonitorSvcType, "monitor-svc", "ClusterIP", "TiDB monitor service type")
	flag.StringVar(&Context.pprofAddr, "pprof", "0.0.0.0:8080", "Pprof address")
	flag.DurationVar(&Context.WaitClusterReadyDuration, "wait-duration", 4*time.Hour, "clusters ready wait duration")

	flag.BoolVar(&Context.Purge, "purge", false, "purge the whole cluster on success")
	flag.BoolVar(&Context.DeleteNS, "delNS", false, "delete the deployed namespace")

	flag.StringVar(&Context.LokiAddress, "loki-addr", "", "loki address. If empty then don't query logs from loki.")
	flag.StringVar(&Context.LokiUsername, "loki-username", "", "loki username. Needed when basic auth is configured in loki")
	flag.StringVar(&Context.LokiPassword, "loki-password", "", "loki password. Needed when basic auth is configured in loki")

	flag.StringVar(&Context.HubAddress, "hub", "", "hub address, default to docker hub")
	flag.StringVar(&Context.TiDBClusterConfig.TiDBHubAddress, "tidb-hub", "", "tidb hub address, will overwrite -hub")
	flag.StringVar(&Context.TiDBClusterConfig.TiKVHubAddress, "tikv-hub", "", "tikv hub address, will overwrite -hub")
	flag.StringVar(&Context.TiDBClusterConfig.PDHubAddress, "pd-hub", "", "pd hub address, will overwrite -hub")
	flag.StringVar(&Context.TiDBClusterConfig.TiFlashHubAddress, "tiflash-hub", "", "tiflash hub address, will overwrite -hub")

	flag.StringVar(&Context.TiDBClusterConfig.ImageVersion, "image-version", "nightly", "image version")
	flag.StringVar(&Context.TiDBClusterConfig.TiDBImage, "tidb-image", "", "tidb image")
	flag.StringVar(&Context.TiDBClusterConfig.TiKVImage, "tikv-image", "", "tikv image")
	flag.StringVar(&Context.TiDBClusterConfig.PDImage, "pd-image", "", "pd image")
	flag.StringVar(&Context.TiDBClusterConfig.TiFlashImage, "tiflash-image", "", "tiflash image")

	flag.StringVar(&Context.TiDBClusterConfig.TiDBConfig, "tidb-config", "", "path of tidb config file (cluster A in abtest case)")
	flag.StringVar(&Context.TiDBClusterConfig.TiKVConfig, "tikv-config", "", "path of tikv config file (cluster A in abtest case)")
	flag.StringVar(&Context.TiDBClusterConfig.PDConfig, "pd-config", "", "path of pd config file (cluster A in abtest case)")
	flag.IntVar(&Context.TiDBClusterConfig.TiDBReplicas, "tidb-replicas", 2, "number of tidb replicas")
	flag.IntVar(&Context.TiDBClusterConfig.TiKVReplicas, "tikv-replicas", 3, "number of tikv replicas")
	flag.IntVar(&Context.TiDBClusterConfig.PDReplicas, "pd-replicas", 3, "number of pd replicas")
	flag.IntVar(&Context.TiDBClusterConfig.TiFlashReplicas, "tiflash-replicas", 0, "number of tiflash replicas, set 0 to disable tiflash")

	flag.StringVar(&Context.ABTestConfig.ClusterBConfig.ImageVersion, "abtest.image-version", "", "specify version for cluster B")
	flag.StringVar(&Context.ABTestConfig.ClusterBConfig.TiDBConfig, "abtest.tidb-config", "", "tidb config file for cluster B")
	flag.StringVar(&Context.ABTestConfig.ClusterBConfig.TiKVConfig, "abtest.tikv-config", "", "tikv config file for cluster B")
	flag.StringVar(&Context.ABTestConfig.ClusterBConfig.PDConfig, "abtest.pd-config", "", "pd config file for cluster B")
	flag.IntVar(&Context.ABTestConfig.ClusterBConfig.TiKVReplicas, "abtest.tikv-replicas", 3, "number of tikv replicas for cluster B")
	flag.IntVar(&Context.ABTestConfig.ClusterBConfig.TiFlashReplicas, "abtest.tiflash-replicas", 0, "number of tiflash replicas for cluster B, set 0 to disable tiflash")

	flag.StringVar(&Context.ABTestConfig.LogPath, "abtest.log", "", "log path for abtest, default to stdout")
	flag.IntVar(&Context.ABTestConfig.Concurrency, "abtest.concurrency", 3, "test concurrency, parallel session number")
	flag.BoolVar(&Context.ABTestConfig.GeneralLog, "abtest.general-log", false, "enable general log in TiDB")

	flag.StringVar(&Context.CDCConfig.Image, "cdc.version", "", `overwrite "-image-version" flag for CDC`)
	flag.StringVar(&Context.CDCConfig.LogPath, "cdc.log", "", "log path for cdc test, default to stdout")
	flag.BoolVar(&Context.CDCConfig.EnableKafka, "cdc.enable-kafka", false, "enable kafka sink")
	flag.StringVar(&Context.CDCConfig.KafkaConsumerImage, "cdc.kafka-consumer-image", "docker.io/pingcap/ticdc-kafka:nightly", "the kafka consumer image to use when kafka is enabled")
	flag.StringVar(&Context.CDCConfig.LogLevel, "cdc.log-level", "debug", "log level for cdc test, default debug")
	flag.StringVar(&Context.CDCConfig.Timezone, "cdc.timezone", "UTC", "timezone of cdc cluster, default UTC")

	flag.StringVar(&Context.DMConfig.MySQLConf.Version, "dm.mysql.version", "5.7", "MySQL version used in DM-pocket")
	flag.StringVar(&Context.DMConfig.MySQLConf.StorageSize, "dm.mysql.storage-size", "10Gi", "request storage size for MySQL")
	flag.BoolVar(&Context.DMConfig.MySQLConf.EnableBinlog, "dm.mysql.enable-binlog", true, "enable binlog for MySQL")
	flag.BoolVar(&Context.DMConfig.MySQLConf.EnableGTID, "dm.mysql.enable-gtid", true, "enable GTID for MySQL")
	flag.StringVar(&Context.DMConfig.DMVersion, "dm.version", "nightly", "image version for DM")
	flag.IntVar(&Context.DMConfig.MasterReplica, "dm.master-replicas", 3, "number of DM-master replicas")
	flag.IntVar(&Context.DMConfig.WorkerReplica, "dm.worker-replicas", 3, "number of DM-worker replicas")

	flag.StringVar(&Context.TiFlashConfig.LogPath, "tiflash.log", "", "log path for TiFlash test, default to stdout")

	flag.BoolVar(&Context.BinlogConfig.EnableRelayLog, "relay-log", false, "if enable relay log")
	flag.StringVar(&Context.BinlogConfig.Image, "binlog-image", "", `overwrite "-image-version" flag for drainer`)
	flag.DurationVar(&Context.BinlogConfig.SyncTimeout, "binlog.sync-timeout", time.Hour, "binlog-like job's sync timeout")

	flag.BoolVar(&Context.EnableHint, "enable-hint", false, "enable to generate sql hint")

	flag.StringVar(&Context.LogPath, "log-path", "/var/run/tipocket-logs", "TiDB cluster logs path")

	// plugins
	flag.StringVar(&Context.LeakCheckEatFile, "plugin.leak.eat", "", "leak check eat file path")
	flag.BoolVar(&Context.LeakCheckSilent, "plugin.leak.silent", true, "leak check silent mode")
	// failpoint
	flag.StringVar(&Context.TiDBFailpoint, "failpoint.tidb", "github.com/pingcap/tidb/server/enableTestAPI=return", "TiDB failpoints")

	flag.Var(&Context.TiDBClusterConfig.TiDBAddr, "tidb-server", "tidb-server addresses")
	flag.Var(&Context.TiDBClusterConfig.TiKVAddr, "tikv-server", "tikv-server addresses")
	flag.Var(&Context.TiDBClusterConfig.PDAddr, "pd-server", "pd-server addresses")

	flag.StringVar(&Context.TiDBClusterConfig.MatrixConfig.MatrixConfigFile, "matrix-config", "", "Matrix config")
	flag.StringVar(&Context.TiDBClusterConfig.MatrixConfig.MatrixTiDBConfig, "matrix-tidb", "", "TiDB config generated by Matrix")
	flag.StringVar(&Context.TiDBClusterConfig.MatrixConfig.MatrixTiKVConfig, "matrix-tikv", "", "TiKV config generated by Matrix")
	flag.StringVar(&Context.TiDBClusterConfig.MatrixConfig.MatrixPDConfig, "matrix-pd", "", "PD config generated by Matrix")
	flag.Var(&Context.TiDBClusterConfig.MatrixConfig.MatrixSQLConfig, "matrix-sql", "SQL files generated by Matrix")
	flag.BoolVar(&Context.TiDBClusterConfig.MatrixConfig.NoCleanup, "no-cleanup-matrix", false, "Do not cleanup Matrix context after initialized")

	log.SetHighlighting(false)
	go func() {
		http.ListenAndServe(Context.pprofAddr, nil)
	}()
}
