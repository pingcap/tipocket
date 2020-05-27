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
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb-tools/pkg/dbutil"
	"github.com/pingcap/tidb-tools/pkg/diff"
	"k8s.io/apimachinery/pkg/util/wait"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	pocketCore "github.com/pingcap/tipocket/pkg/pocket/core"
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

// dmSyncDiffData uses sync-diff-inspector to check data between downstream and upstream for DM tasks.
func dmSyncDiffData(ctx context.Context, pCore *pocketCore.Core, checkInterval time.Duration, clientNodes []clusterTypes.ClientNode) error {
	schema := "pocket" // we always use `pocket` as the DB name now, see `makeDSN`.
	mysql1 := clientNodes[0]
	tidb := clientNodes[2]
	mysql1Cfg := dbutil.DBConfig{
		Host:   mysql1.IP,
		Port:   int(mysql1.Port),
		User:   "root",
		Schema: schema,
	}
	tidbCfg := dbutil.DBConfig{
		Host:   tidb.IP,
		Port:   int(tidb.Port),
		User:   "root",
		Schema: schema,
	}
	mysql1DB, err := dbutil.OpenDB(mysql1Cfg)
	if err != nil {
		return err
	}
	tidbDB, err := dbutil.OpenDB(tidbCfg)
	if err != nil {
		return err
	}

	go func() { // no wait for it to return
		for range time.Tick(checkInterval) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			pCore.ExecLock()
			err := wait.PollImmediate(30*time.Second, 5*time.Minute, func() (done bool, err error) {
				if err2 := dmSyncDiffSingleTask(ctx, mysql1DB, tidbDB, schema); err2 != nil {
					log.Errorf("fail to diff data for single source task, %v", err)
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				log.Fatalf("fail to diff data for single source task, %v", err)
			}
			log.Info("consistency check pass for DM single source task")
			pCore.ExecUnlock()
		}
	}()
	return nil
}

// dmSyncDiffSingleTask checks data for the single source task.
func dmSyncDiffSingleTask(ctx context.Context, mysqlDB, tidbDB *sql.DB, schema string) error {
	log.Infof("ready to compare data for the single source task")
	// get all tables.
	tables, err := dbutil.GetTables(ctx, mysqlDB, schema)
	if err != nil {
		return errors.Errorf("fail to get tables for schema %s, %v", schema, err)
	}

	for _, table := range tables {
		sourceTables := []*diff.TableInstance{
			{
				Conn:       mysqlDB,
				Schema:     schema,
				Table:      table,
				InstanceID: "source-1",
			},
		}

		targetTable := &diff.TableInstance{
			Conn:       tidbDB,
			Schema:     schema,
			Table:      table,
			InstanceID: "target",
		}

		td := &diff.TableDiff{
			SourceTables:     sourceTables,
			TargetTable:      targetTable,
			ChunkSize:        1000,
			Sample:           100,
			CheckThreadCount: 1,
			UseChecksum:      true,
			TiDBStatsSource:  targetTable,
			CpDB:             tidbDB,
		}

		structEqual, dataEqual, err := td.Equal(ctx, func(dml string) error {
			fmt.Println("fix SQL:", dml) // output to stdout
			return nil
		})

		if errors.Cause(err) == context.Canceled || errors.Cause(err) == context.DeadlineExceeded {
			return nil
		}
		if !structEqual {
			return errors.Errorf("different struct for table %s", table)
		} else if !dataEqual {
			return errors.Errorf("different data for table %s", table)
		}
		log.Infof("struct and data are equal for table %s", table)
	}

	return nil
}
