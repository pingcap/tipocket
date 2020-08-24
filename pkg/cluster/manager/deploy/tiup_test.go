package deploy

import (
	"testing"

	"github.com/alecthomas/assert"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

func TestBuildTopologyYaml(t *testing.T) {
	yaml := buildTopologyYaml(&Topology{
		Config:            `server_configs:
  tidb:
    log.slow-threshold: 300
    binlog.enable: false
    binlog.ignore-error: false
  tikv:
    # server.grpc-concurrency: 4
    # raftstore.apply-pool-size: 2
    # raftstore.store-pool-size: 2
    # rocksdb.max-sub-compactions: 1
    # storage.block-cache.capacity: "16GB"
    # readpool.unified.max-thread-count: 12
    readpool.storage.use-unified-pool: false
    readpool.coprocessor.use-unified-pool: true
  pd:
    schedule.leader-schedule-limit: 4
    schedule.region-schedule-limit: 2048
    schedule.replica-schedule-limit: 64
  tiflash:
    # path_realtime_mode: false
    logger.level: "info"
  tiflash-learner:
    log-level: "info"
    # raftstore.apply-pool-size: 4
    # raftstore.store-pool-size: 4
  pump:
    gc: 7
`,
		PDServers: map[string]*types.ClusterRequestTopology{
			"10.0.1.11": &types.ClusterRequestTopology{
				Component:  "pd",
				DeployPath: "/data1",
			},
			"10.0.1.12": &types.ClusterRequestTopology{
				Component:  "pd",
				DeployPath: "/data1",
			},
		},
		TiKVServers: map[string]*types.ClusterRequestTopology{
			"10.0.1.13": &types.ClusterRequestTopology{
				Component:  "tikv",
				DeployPath: "/data1",
			},
			"10.0.1.14": &types.ClusterRequestTopology{
				Component:  "tikv",
				DeployPath: "/data1",
			},
		},
		TiDBServers: map[string]*types.ClusterRequestTopology{
			"10.0.1.15": &types.ClusterRequestTopology{
				Component:  "tidb",
				DeployPath: "/data2",
			},
		},
		MonitoringServers: map[string]*types.ClusterRequestTopology{
			"10.0.1.16": &types.ClusterRequestTopology{
				Component:  "tidb",
				DeployPath: "/data2",
			},
		},
		GrafanaServers: map[string]*types.ClusterRequestTopology{
			"10.0.1.17": &types.ClusterRequestTopology{
				Component:  "tidb",
				DeployPath: "/data2",
			},
		},
	})

	assert.Equal(t, `global:
  user: "tidb"
  ssh_port: 22
  arch: "amd64"

server_configs:
  tidb:
    log.slow-threshold: 300
    binlog.enable: false
    binlog.ignore-error: false
  tikv:
    # server.grpc-concurrency: 4
    # raftstore.apply-pool-size: 2
    # raftstore.store-pool-size: 2
    # rocksdb.max-sub-compactions: 1
    # storage.block-cache.capacity: "16GB"
    # readpool.unified.max-thread-count: 12
    readpool.storage.use-unified-pool: false
    readpool.coprocessor.use-unified-pool: true
  pd:
    schedule.leader-schedule-limit: 4
    schedule.region-schedule-limit: 2048
    schedule.replica-schedule-limit: 64
  tiflash:
    # path_realtime_mode: false
    logger.level: "info"
  tiflash-learner:
    log-level: "info"
    # raftstore.apply-pool-size: 4
    # raftstore.store-pool-size: 4
  pump:
    gc: 7


pd_servers:
  - host: 10.0.1.11
    deploy_dir: "/data1/deploy/pd-2379"
    data_dir: "/data1/data/pd-2379"
  - host: 10.0.1.12
    deploy_dir: "/data1/deploy/pd-2379"
    data_dir: "/data1/data/pd-2379"

tidb_servers:
  - host: 10.0.1.15
    deploy_dir: "/data2/deploy/tidb-4000"
    data_dir: "/data2/data/tidb-4000"

tikv_servers:
  - host: 10.0.1.15
    deploy_dir: "/data2/deploy/tikv-20160"
    data_dir: "/data2/data/tikv-20160"

monitoring_servers:
  - host: 10.0.1.16
    deploy_dir: "/data2/data/prometheus-8249"

grafana_servers:
  - host: 10.0.1.17
    deploy_dir: "/data2/data/grafana-3000"`, yaml)
}
