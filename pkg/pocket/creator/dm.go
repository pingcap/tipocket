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

package creator

import (
	"fmt"
	"net/http"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"k8s.io/apimachinery/pkg/util/wait"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/util/dmutil"
)

// create MySQL sources and start tasks.
func dmCreateSourceTask(clientNodes []clusterTypes.ClientNode) error {
	sourceTemp := `
source-id = "%s"
enable-gtid = true

[from]
host = "%s"
user = "root"
port = %d
`

	sourceID1 := "source-1"
	sourceID2 := "source-2"
	mysql1 := clientNodes[0]
	mysql2 := clientNodes[1]
	source1 := fmt.Sprintf(sourceTemp, sourceID1, mysql1.IP, mysql1.Port)
	source2 := fmt.Sprintf(sourceTemp, sourceID2, mysql2.IP, mysql2.Port)

	master := clientNodes[3] // the first DM-master node.
	masterAddr := fmt.Sprintf("%s:%d", master.IP, master.Port)

	log.Infof(`create sources:
master-addr:%s

source1:  ---
%s

source2:  ---
%s
`, masterAddr, source1, source2)

	// use HTTP API to create source.
	client := dmutil.NewDMClient(&http.Client{}, masterAddr)
	if err := client.CreateSource(source1); err != nil {
		return errors.Errorf("fail to create source1: %v", err)
	}
	if err := client.CreateSource(source2); err != nil {
		return errors.Errorf("fail to create source2: %v", err)
	}

	taskSingleTemp := `
name: "%s"
task-mode: "all"

target-database:
  host: "%s"
  port: %d
  user: "root"

mysql-instances:
-
  source-id: "%s"
  black-white-list: "global"

black-white-list:
  global:
    do-dbs: ["%s"]
`

	// use HTTP API to start task.
	taskName := "dm-single"
	tidb := clientNodes[2]
	dbName := "pocket" // we always use `pocket` as the DB name now, see `makeDSN`.
	task := fmt.Sprintf(taskSingleTemp, taskName, tidb.IP, tidb.Port, sourceID1, dbName)

	log.Infof(`start task:
master-addr:%s
%s`, masterAddr, task)

	if err := client.StartTask(task, 1); err != nil {
		return errors.Errorf("fail to start task: %v", err)
	}

	// check task stage is `Running`.
	err := wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
		err := client.CheckTaskStage(taskName, "Running", 1)
		if err != nil {
			log.Errorf("fail to check task stage: %v", err)
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return errors.Errorf("fail to check task stage: %v", err)
	}

	return nil
}
