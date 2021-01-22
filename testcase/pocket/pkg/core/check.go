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

package core

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/executor"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/util"
)

func (c *Core) startCheckConsistency(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.Options.CheckDuration.Duration)
	round := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var (
				result bool
				err    error
			)
			log.Info("ready to compare data")

			err = wait.PollImmediate(1*time.Minute, 10*time.Minute, func() (bool, error) {
				result, err = c.checkConsistency(false)
				if err != nil {
					// TODO: pass error by channel, stop process from outside
					log.Errorf("compare data error %+v", errors.ErrorStack(err))
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				log.Errorf("compare data failed %v", err)
				return
			}
			log.Infof("test %d compare data result %t\n", round, result)
			round++
		}
	}
}

func (c *Core) checkConsistency(delay bool) (bool, error) {
	var (
		result bool
		err    error
	)
	switch c.cfg.Mode {
	case "abtest", "tiflash-abtest":
		result, err = c.abTestCompareData(delay)
	case "binlog", "tiflash-binlog":
		result, err = c.binlogTestCompareData(delay)
	default:
		result, err = true, nil
	}
	return result, errors.Trace(err)
}

// abTestCompareDataWithoutCommit take snapshot without other transactions all committed
// this function can run async and channel is for waiting taking snapshot complete
func (c *Core) abTestCompareDataWithoutCommit(ch chan struct{}) {
	// start a temp session for keep the snapshot of state
	compareExecutor, err := c.initCompareConnection()
	if err != nil {
		log.Fatal(err)
	}
	// schema should be fetch first
	schema, err := compareExecutor.GetConn().FetchSchema(c.dbname)
	if err != nil {
		log.Fatal(err)
	}
	if err := compareExecutor.ABTestTxnBegin(); err != nil {
		log.Fatal(err)
	}
	sqls := makeCompareSQLs(schema)
	if err := compareExecutor.ABTestSelect(sqls[0]); err != nil {
		log.Fatal(err)
	}
	begin := util.CurrentTimeStrAsLog()
	ch <- struct{}{}

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	if err != nil {
		log.Fatal("get schema err %+v", errors.ErrorStack(err))
	}
	for _, sql := range sqls {
		if err := compareExecutor.ABTestSelect(sql); err != nil {
			log.Fatalf("inconsistency when exec %s compare data %+v, begin: %s\n", sql, err, begin)
		}
	}
	log.Info("consistency check pass")
}

func (c *Core) abTestCompareData(delay bool) (bool, error) {
	// start a temp session for keep the snapshot of state
	compareExecutor, err := c.initCompareConnection()
	if err != nil {
		return false, errors.Trace(err)
	}
	defer func(compareExecutor *executor.Executor) {
		if err := compareExecutor.Close(); err != nil {
			log.Fatal("close compare executor error %+v\n", errors.ErrorStack(err))
		}
	}(compareExecutor)

	// commit or rollback all transactions
	log.Info("before lock")
	c.execMutex.Lock()
	c.Lock()
	log.Info("after lock")
	// no async here to ensure all transactions are committed or rollbacked in order
	// use resolveDeadLock func to avoid deadlock
	c.resolveDeadLock(true)
	schema, err := compareExecutor.GetConn().FetchSchema(c.dbname)
	if err != nil {
		c.Unlock()
		c.execMutex.Unlock()
		return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestTxnBegin(); err != nil {
		c.Unlock()
		c.execMutex.Unlock()
		return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestSelect(makeCompareSQLs(schema)[0]); err != nil {
		if errors.Cause(err) == util.ErrExactlyNotSame {
			log.Fatal(err)
		}
		return false, errors.Trace(err)
	}
	// free the lock since the compare has already got the same snapshot in both side
	// go on other transactions
	// defer can be removed
	// but here we use it for protect environment
	defer func() {
		log.Info("free lock")
		c.Unlock()
		c.execMutex.Unlock()
	}()

	// delay will hold on this snapshot and check it later
	if delay {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}

	return c.compareData(compareExecutor, schema)
}

func (c *Core) binlogTestCompareData(delay bool) (bool, error) {
	// start a temp session for keep the snapshot of state
	compareExecutor, err := c.initCompareConnection()
	if err != nil {
		return false, errors.Trace(err)
	}
	defer func(compareExecutor *executor.Executor) {
		if err := compareExecutor.Close(); err != nil {
			log.Errorf("close compare executor error %+v\n", errors.ErrorStack(err))
		}
	}(compareExecutor)

	// commit or rollback all transactions
	// lock here before get snapshot
	c.execMutex.Lock()
	c.Lock()
	// free the lock since the compare has already got the same snapshot in both side
	// go on other transactions
	// defer can be removed
	// but here we use it for protect environment
	defer func() {
		log.Info("free lock")
		c.Unlock()
		c.execMutex.Unlock()
	}()

	// no async here to ensure all transactions are committed or rollbacked in order
	// use resolveDeadLock func to avoid deadlock
	c.resolveDeadLock(true)

	// insert a table and wait for the sync job is done
	table, tableStmt := generateWaitTable()
	for compareExecutor.SingleTestExecDDL(tableStmt) != nil {
		time.Sleep(time.Second)
	}
	var (
		ctx, cancel = context.WithTimeout(context.Background(), c.cfg.Options.SyncTimeout.Duration)
		ticker      = time.NewTicker(10 * time.Second)
	)
	defer cancel()

SYNC:
	for {
		select {
		case <-ctx.Done():
			return false, errors.Errorf("sync timeout in %s", c.cfg.Options.SyncTimeout.Duration)
		case <-ticker.C:
			tables, err := compareExecutor.GetConn2().FetchTables(c.dbname)
			if err != nil {
				// not throw error here
				// may be caused by chaos
				// we should wait for sync timeout before throwing an error
				log.Error(err)
			}
			for _, t := range tables {
				if t == table {
					break SYNC
				}
			}
		}
	}

	schema, err := compareExecutor.GetConn().FetchSchema(c.dbname)
	for err != nil {
		schema, err = compareExecutor.GetConn().FetchSchema(c.dbname)
		// c.Unlock()
		// return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestTxnBegin(); err != nil {
		return false, errors.Trace(err)
	}
	log.Info("compare wait for chan finish")
	// delay will hold on this snapshot and check it later
	if delay {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}

	return c.compareData(compareExecutor, schema)
}

func (c *Core) compareData(beganConnect *executor.Executor, schema [][5]string) (bool, error) {
	sqls := makeCompareSQLs(schema)
	for _, sql := range sqls {
		err := wait.PollImmediate(1*time.Minute, 10*time.Minute, func() (done bool, err error) {
			if err := beganConnect.ABTestSelect(sql); err != nil {
				if errors.Cause(err) == util.ErrExactlyNotSame {
					log.Fatalf("inconsistency when exec %s compare data %+v, begin: %s\n",
						sql, err, util.FormatTimeStrAsLog(beganConnect.GetConn().GetBeginTime()))
				}
				log.Errorf("a/b testing occurred an error and will retry later, %+v", err)
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return false, errors.Trace(err)
		}
	}
	log.Info("consistency check pass")
	return true, nil
}

func makeCompareSQLs(schema [][5]string) []string {
	rowCountSQLs := []string{}
	columnDataSQLs := []string{}
	tables := make(map[string][]string)

	for _, record := range schema {
		if _, ok := tables[record[1]]; !ok {
			tables[record[1]] = []string{}
		}
		if record[3] != "id" {
			tables[record[1]] = append(tables[record[1]], record[3])
		}
	}

	for name, table := range tables {
		rowCountSQLs = append(rowCountSQLs, fmt.Sprintf("SELECT COUNT(1) FROM %s", name))
		columnDataSQL := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(table, ", "), name, strings.Join(table, ", "))
		columnDataSQLs = append(columnDataSQLs, columnDataSQL)
	}

	sort.Sort(types.BySQL(rowCountSQLs))
	sort.Sort(types.BySQL(columnDataSQLs))
	return append(rowCountSQLs, columnDataSQLs...)
}

func generateWaitTable() (string, string) {
	sec := time.Now().Unix()
	table := fmt.Sprintf("t%d", sec)
	return table, fmt.Sprintf("CREATE TABLE %s(id int PRIMARY KEY)", table)
}
