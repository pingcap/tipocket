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

package binlog

import (
	"bytes"
	"html/template"
)

var drainerConfigTpl = template.Must(template.New("drainer-config-script").Parse(`# drainer Configuration.

# addr (i.e. 'host:port') to listen on for drainer connections
# will register this addr into etcd
# addr = "127.0.0.1:8249"

# the interval time (in seconds) of detect pumps' status
detect-interval = 10

# drainer meta data directory path
data-dir = "/data"

# a comma separated list of PD endpoints
pd-urls = "http://{{.PDAddress}}:2379"

# Use the specified compressor to compress payload between pump and drainer
compressor = ""

#[security]
# Path of file that contains list of trusted SSL CAs for connection with cluster components.
# ssl-ca = "/path/to/ca.pem"
# Path of file that contains X509 certificate in PEM format for connection with cluster components.
# ssl-cert = "/path/to/pump.pem"
# Path of file that contains X509 key in PEM format for connection with cluster components.
# ssl-key = "/path/to/pump-key.pem"

# syncer Configuration.
[syncer]

# Assume the upstream sql-mode.
# If this is set , will use the same sql-mode to parse DDL statment, and set the same sql-mode at downstream when db-type is mysql.
# The default value will not set any sql-mode.
# sql-mode = "STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION"

# number of binlog events in a transaction batch
txn-batch = 20

# work count to execute binlogs
# if the latency between drainer and downstream(mysql or tidb) are too high, you might want to increase this
# to get higher throughput by higher concurrent write to the downstream
worker-count = 16

disable-dispatch = false

# safe mode will split update to delete and insert
safe-mode = false

# downstream storage, equal to --dest-db-type
# valid values are "mysql", "pb", "tidb", "flash", "kafka"
db-type = "mysql"

# disable sync these schema
ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql,test"

##replicate-do-db priority over replicate-do-table if have same db name
##and we support regex expression , start with '~' declare use regex expression.
#
#replicate-do-db = ["~^b.*","s1"]

#[[syncer.replicate-do-table]]
#db-name ="test"
#tbl-name = "log"

#[[syncer.replicate-do-table]]
#db-name ="test"
#tbl-name = "~^a.*"

# disable sync these table
#[[syncer.ignore-table]]
#db-name = "test"
#tbl-name = "log"
# the downstream mysql protocol database
[syncer.to]
host = "{{.DownStreamDB}}"
user = "root"
password = ""
port = 4000

[syncer.to.checkpoint]
# you can uncomment this to change the database to save checkpoint when the downstream is mysql or tidb
#schema = "tidb_binlog"`))

type DrainerConfigModel struct {
	PDAddress    string
	DownStreamDB string
}

func RenderDrainerConfig(model *DrainerConfigModel) (string, error) {
	return renderTemplateFunc(drainerConfigTpl, model)
}

var drainerCommandTpl = template.Must(template.New("drainer-command").Parse(`set -euo pipefail

domain=` + "`" + `echo ${HOSTNAME}` + "`" + `.{{.Component}}

elapseTime=0
period=1
threshold=30
while true; do
    sleep ${period}
    elapseTime=$(( elapseTime+period ))

    if [[ ${elapseTime} -ge ${threshold} ]]
    then
        echo "waiting for drainer domain ready timeout" >&2
        exit 1
    fi

    if nslookup ${domain} 2>/dev/null
    then
        echo "nslookup domain ${domain} success"
        break
    else
        echo "nslookup domain ${domain} failed" >&2
    fi
done

/drainer \
-L=debug \
-addr=` + "`" + `echo ${HOSTNAME}` + "`" + `.{{.Component}}:8249 \
-config=/etc/drainer/drainer.toml \
-disable-detect=false \
-initial-commit-ts=0 \
-log-file=/data/log/drainer.log`))

type DrainerCommandModel struct {
	Component string
}

func RenderDrainerCommand(model *DrainerCommandModel) (string, error) {
	return renderTemplateFunc(drainerCommandTpl, model)
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
