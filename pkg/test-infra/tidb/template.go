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

package tidb

import (
	"html/template"

	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// tidbStartScriptTpl is the template string of tidb start script
// Note: changing this will cause a rolling-update of tidb-servers
var tidbStartScriptTpl = template.Must(template.New("tidb-start-script").Parse(`#!/bin/sh

# This script is used to start tidb containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#
set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null
runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--store=tikv \
--advertise-address=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc \
--host=0.0.0.0 \
--path=${CLUSTER_NAME}-pd:2379 \
--config=/etc/tidb/tidb.toml \
--log-file=/var/log/tidblog/tidb.log
"

if [[ X${BINLOG_ENABLED:-} == Xtrue ]]
then
    ARGS="${ARGS} --enable-binlog=true"
fi

SLOW_LOG_FILE=${SLOW_LOG_FILE:-""}
if [[ ! -z "${SLOW_LOG_FILE}" ]]
then
    ARGS="${ARGS} --log-slow-query=${SLOW_LOG_FILE:-}"
fi

if [[ ! -z "{{.Failpoints}}" ]];
then
	export GO_FAILPOINTS='{{.Failpoints}}'
	echo "export GO_FAILPOINTS='{{.Failpoints}}'"
fi
echo "start tidb-server ..."
echo "/tidb-server ${ARGS}"
tail -F /var/log/tidblog/tidb.log &
#    ^^  -F   same as --follow=name --retry, support log rotated
/tidb-server ${ARGS}
`))

// StartScriptModel ...
type StartScriptModel struct {
	ClusterName string
	Failpoints  string
}

// RenderTiDBStartScript ...
func RenderTiDBStartScript(model *StartScriptModel) (string, error) {
	return util.RenderTemplateFunc(tidbStartScriptTpl, model)
}

var pdStartScriptTpl = template.Must(template.New("pd-start-script").Parse(`#!/bin/sh

# This script is used to start pd containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
# the general form of variable PEER_SERVICE_NAME is: "<clusterName>-pd-peer"
cluster_name=` + "`" + `echo ${PEER_SERVICE_NAME} | sed 's/-pd-peer//'` + "`" + "\n" +
	`
domain="${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc" 
discovery_url="${cluster_name}-discovery.${NAMESPACE}.svc:10261"
encoded_domain_url=` + "`" + `echo ${domain}:2380 | base64 | tr "\n" " " | sed "s/ //g"` + "`" +
	`
elapseTime=0
period=1
threshold=30
while true; do
sleep ${period}
elapseTime=$(( elapseTime+period ))

if [[ ${elapseTime} -ge ${threshold} ]]
then
echo "waiting for pd cluster ready timeout" >&2
exit 1
fi

if nslookup ${domain} 2>/dev/null
then
echo "nslookup domain ${domain}.svc success"
break
else
echo "nslookup domain ${domain} failed" >&2
fi
done

ARGS="--data-dir={{.DataDir}} \
--name=${POD_NAME} \
--peer-urls=http://0.0.0.0:2380 \
--advertise-peer-urls=http://${domain}:2380 \
--client-urls=http://0.0.0.0:2379 \
--advertise-client-urls=http://${domain}:2379 \
--config=/etc/pd/pd.toml \
--log-file=/var/log/pdlog/pd.log
"

if [[ -f {{.DataDir}}/join ]]
then
# The content of the join file is:
#   demo-pd-0=http://demo-pd-0.demo-pd-peer.demo.svc:2380,demo-pd-1=http://demo-pd-1.demo-pd-peer.demo.svc:2380
# The --join args must be:
#   --join=http://demo-pd-0.demo-pd-peer.demo.svc:2380,http://demo-pd-1.demo-pd-peer.demo.svc:2380
join=` + "`" + `cat {{.DataDir}}/join | tr "," "\n" | awk -F'=' '{print $2}' | tr "\n" ","` + "`" + `
join=${join%,}
ARGS="${ARGS} --join=${join}"
elif [[ ! -d {{.DataDir}}/member/wal ]]
then
until result=$(wget -qO- -T 3 http://${discovery_url}/new/${encoded_domain_url} 2>/dev/null); do
echo "waiting for discovery service to return start args ..."
sleep $((RANDOM % 5))
done
ARGS="${ARGS}${result}"
fi

echo "starting pd-server ..."
sleep $((RANDOM % 10))
echo "/pd-server ${ARGS}"
tail -F /var/log/pdlog/pd.log &
#    ^^  -F   same as --follow=name --retry, support log rotated
/pd-server ${ARGS}
`))

// PDStartScriptModel ...
type PDStartScriptModel struct {
	DataDir string
}

// RenderPDStartScript ...
func RenderPDStartScript(model *PDStartScriptModel) (string, error) {
	return util.RenderTemplateFunc(pdStartScriptTpl, model)
}

var tikvStartScriptTpl = template.Must(template.New("tikv-start-script").Parse(`#!/bin/sh

# This script is used to start tikv containers in kubernetes cluster

# Use DownwardAPIVolumeFiles to store informations of the cluster:
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#the-downward-api
#
#   runmode="normal/debug"
#

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

runmode=${runmode:-normal}
if [[ X${runmode} == Xdebug ]]
then
	echo "entering debug mode."
	tail -f /dev/null
fi

# support for data-encryption
if [[ "{{.MasterKey}}" != "" ]]
then
	echo "writing master key {{.MasterKey}} to /var/lib/tikv/master_key"
	echo "{{.MasterKey}}" > /var/lib/tikv/master_key
fi

# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--pd=http://${CLUSTER_NAME}-pd:2379 \
--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:20160 \
--addr=0.0.0.0:20160 \
--status-addr=0.0.0.0:20180 \
--data-dir={{.DataDir}} \
--capacity=${CAPACITY} \
--config=/etc/tikv/tikv.toml \
--log-file=/var/lib/tikv/tikvlog/tikv.log
"

# Oops, i put tikv log directory with data together for reducing PV.
mkdir -p /var/lib/tikv/tikvlog
echo "starting tikv-server ..."
echo "/tikv-server ${ARGS}"
tail -F /var/lib/tikv/tikvlog/tikv.log &
#    ^^  -F   same as --follow=name --retry, support log rotated
/tikv-server ${ARGS}
`))

// TiKVStartScriptModel ...
type TiKVStartScriptModel struct {
	DataDir   string
	MasterKey string
}

// RenderTiKVStartScript ...
func RenderTiKVStartScript(model *TiKVStartScriptModel) (string, error) {
	return util.RenderTemplateFunc(tikvStartScriptTpl, model)
}

var pumpConfigTpl = template.Must(template.New("pump-config-script").Parse(`detect-interval = 10
compressor = ""
[syncer]
worker-count = 16
disable-dispatch = false
ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"
safe-mode = false
txn-batch = 20
db-type = "file"
[syncer.to]
dir = "/data/pb"`))

type pumpConfigModel struct {
}

// RenderPumpConfig ...
func RenderPumpConfig(model *pumpConfigModel) (string, error) {
	return util.RenderTemplateFunc(pumpConfigTpl, model)
}
