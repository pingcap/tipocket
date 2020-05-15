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

package executor

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ngaut/log"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tipocket/pkg/pocket/connection"
	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
	"github.com/pingcap/tipocket/pkg/pocket/util"
	"github.com/pingcap/tipocket/pkg/util/dmutil"
)

func (e *Executor) dmTest() {
	// create source for DM upstream MySQL.
	e.dmCreateSource()
	for {
		e.ErrCh <- e.execDMTestSQL(<-e.SQLCh)
	}
}

func (e *Executor) dmCreateSource() {
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
	mysql1 := e.opt.Cfg.ClientNodes[0]
	mysql2 := e.opt.Cfg.ClientNodes[1]
	source1 := fmt.Sprintf(sourceTemp, sourceID1, mysql1.IP, mysql1.Port)
	source2 := fmt.Sprintf(sourceTemp, sourceID2, mysql2.IP, mysql2.Port)

	master := e.opt.Cfg.ClientNodes[3] // the first DM-master node.
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
		panic(fmt.Sprintf("fail to create source1: %v", err))
	}
	if err := client.CreateSource(source2); err != nil {
		panic(fmt.Sprintf("fail to create source2: %v", err))
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
	tidb := e.opt.Cfg.ClientNodes[2]
	task := fmt.Sprintf(taskSingleTemp, taskName, tidb.IP, tidb.Port, sourceID1, e.dbname)

	log.Infof(`start task:
master-addr:%s
%s`, masterAddr, task)

	if err := client.StartTask(task, 1); err != nil {
		panic(fmt.Sprintf("fail to start task: %v", err))
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
		panic(fmt.Sprintf("fail to check task stage: %v", err))
	}
}

func (e *Executor) execDMTestSQL(sql *types.SQL) error {
	e.logStmtTodo(sql.SQLStmt)
	var err error
	switch sql.SQLType {
	case types.SQLTypeDMLInsert, types.SQLTypeDMLUpdate, types.SQLTypeDMLDelete,
		types.SQLTypeTxnBegin, types.SQLTypeTxnCommit, types.SQLTypeTxnRollback,
		types.SQLTypeCreateDatabase, types.SQLTypeDropDatabase,
		types.SQLTypeDDLCreateTable, types.SQLTypeDDLAlterTable,
		types.SQLTypeDDLCreateIndex,
		types.SQLTypeExec, types.SQLTypeSleep:
		err = e.execDMTestStmt(sql.SQLStmt)
	case types.SQLTypeExit:
		e.Stop("receive exit SQL signal")
	default:
		log.Debugf("ignore SQL %v", sql)
	}

	e.logStmtResult(sql.SQLStmt, err)
	return err
}

func (e *Executor) execDMTestStmt(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)

	wg.Add(2)
	go func() {
		err1 = e.conn1.Exec(sql)
		wg.Done()
	}()
	go func() {
		// TODO(csuzhangxc): shard merge support
		err2 = e.conn2.Exec(sql)
		wg.Done()
	}()
	wg.Wait()

	if err := util.ErrorMustSame(err1, err2); err != nil {
		return err
	}
	return err1
}

// DMSelectEqual check data equal by `SELECT`.
func (e *Executor) DMSelectEqual(sql string, conn1, conn2 *connection.Connection) error {
	var (
		wg   sync.WaitGroup
		res1 [][]*connection.QueryItem
		res2 [][]*connection.QueryItem
		err1 error
		err2 error
	)

	wg.Add(2)
	go func() {
		res1, err1 = conn1.Select(sql)
		wg.Done()
	}()
	go func() {
		res2, err2 = conn2.Select(sql)
		wg.Done()
	}()
	wg.Wait()

	if err1 != nil {
		return err1
	} else if err2 != nil {
		return err2
	}

	if len(res1) != len(res2) {
		return util.WrapErrExactlyNotSame("row number not match res1: %d, res2: %d", len(res1), len(res2))
	}
	for index := range res1 {
		var (
			row1 = res1[index]
			row2 = res2[index]
		)

		if len(row1) != len(row1) {
			return util.WrapErrExactlyNotSame("column number not match res1: %d, res2: %d", len(res1), len(res2))
		}

		for rIndex := range row1 {
			var (
				item1 = row1[rIndex]
				item2 = row2[rIndex]
			)
			if err := item1.MustSame(item2); err != nil {
				return util.WrapErrExactlyNotSame("%s, row index %d, column index %d", err.Error(), index, rIndex)
			}
		}
	}

	return nil
}

func (e *Executor) dmTestIfTxn() bool {
	return e.conn1.IfTxn() || e.conn2.IfTxn()
}
