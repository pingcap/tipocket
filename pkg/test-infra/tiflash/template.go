// Copyright 2020 PingCAP, Inc.
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

package tiflash

import (
	"html/template"

	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

const (
	usersTemplate = `[quotas]

[quotas.default]

[quotas.default.interval]
duration = 3600
queries = 0
errors = 0
result_rows = 0
read_rows = 0
execution_time = 0

[users]

[users.readonly]
password = ""
profile = "readonly"
quota = "default"

[users.readonly.networks]
ip = "::/0"

[users.default]
password = ""
profile = "default"
quota = "default"

[users.default.networks]
ip = "::/0"

[profiles]

[profiles.readonly]
readonly = 1

[profiles.default]
max_memory_usage = 10000000000
use_uncompressed_cache = 0
load_balancing = "random"
`

	tiFlashInitCmdTemplate = `set -ex
          [[ ${HOSTNAME} =~ -([0-9]+)$ ]] || exit 1
          ordinal=${BASH_REMATCH[1]}
          sed s/{pod_num}/${ordinal}/g /etc/tiflash/config_templ.toml > /data/config.toml
          sed s/{pod_num}/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data/proxy.toml
	`
)

type tiFlashConfig struct {
	ClusterName string
	Namespace   string
}

func renderTiFlashConfig(model *tiFlashConfig) (string, error) {
	return util.RenderTemplateFunc(tiFlashTpl, model)
}

var tiFlashTpl = template.Must(template.New("tiflash-config").Parse(`
tmp_path = "/data/tmp"
display_name = "TiFlash"
default_profile = "default"
users_config = "/etc/tiflash/users.toml"
path = "/data/db"
mark_cache_size = 5368709120
listen_host = "0.0.0.0"
tcp_port = 9000
http_port = 8123
interserver_http_port = 9009

[flash]
tidb_status_addr = "{{.ClusterName}}-tidb.{{.Namespace}}.svc.cluster.local:10080"
service_addr = "{{.ClusterName}}-tiflash-{pod_num}.{{.ClusterName}}-tiflash.{{.Namespace}}.svc.cluster.local:3930"
overlap_threshold = 0.6
compact_log_min_period = 10

[flash.flash_cluster]
master_ttl = 60
refresh_interval = 20
update_rule_interval = 5
cluster_manager_path = "/flash_cluster_manager"

[flash.proxy]
addr = "0.0.0.0:20170"
data-dir = "/data/proxy"
config = "/data/proxy.toml"
log-file = "/data/logs/proxy.log"

[logger]
level = "debug"
log = "/data/logs/server.log"
errorlog = "/data/logs/error.log"
size = "4000M"
count = 10

[application]
runAsDaemon = true

[raft]
kvstore_path = "/data/kvstore"
pd_addr = "{{.ClusterName}}-pd.{{.Namespace}}.svc.cluster.local:2379"
storage_engine = "dt"

[status]
metrics_port = 8234

`))

func renderTiFlashProxyTpl(model *tiFlashConfig) (string, error) {
	return util.RenderTemplateFunc(tiFlashProxyTpl, model)
}

var tiFlashProxyTpl = template.Must(template.New("tiflash-proxy").Parse(`
log-level = "info"

[readpool.storage]

[readpool.coprocessor]

[server]
engine-addr = "{{.ClusterName}}-tiflash-{pod_num}.{{.ClusterName}}-tiflash.{{.Namespace}}.svc.cluster.local:3930"
advertise-addr = "{{.ClusterName}}-tiflash-{pod_num}.{{.ClusterName}}-tiflash.{{.Namespace}}.svc.cluster.local:20170"

[storage]

[pd]

[metric]

[raftstore]
raftdb-path = ""
sync-log = true
#max-leader-missing-duration = "22s"
#abnormal-leader-missing-duration = "21s"
#peer-stale-state-check-interval = "20s"
hibernate-regions = false

[coprocessor]

[rocksdb]
wal-dir = ""
max-open-files = 1000

[rocksdb.defaultcf]
block-cache-size = "10GB"

[rocksdb.lockcf]
block-cache-size = "4GB"

[rocksdb.writecf]
block-cache-size = "4GB"

[raftdb]
max-open-files = 1000

[raftdb.defaultcf]
block-cache-size = "1GB"

[security]
ca-path = ""
cert-path = ""
key-path = ""

[import]

`))
