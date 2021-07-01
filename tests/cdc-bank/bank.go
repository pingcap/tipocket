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

package cdcbank

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config contains the configurations of the CDC bank case
type Config struct {
	Accounts    int
	Concurrency int
	RunTime     time.Duration
}

// ClientCreator is a CDC bank creator
type ClientCreator struct {
	Cfg *Config
}

// Create creates a CDC bank client
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &client{
		Config: c.Cfg,
	}
}

const initBalance int = 1000
const regions int = 16
const validateConcurrency int = 32

type client struct {
	*Config
}

func pk(accountID int) int {
	return accountID * 10000
}

func (c *client) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	var upstream, downstream *sql.DB
	expectedSum := c.Accounts * initBalance

	for i, node := range clientNodes {
		dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
		if i == 0 {
			// upstream TiDB
			log.Infof("set up upstream DB connection, dsn: %v", dsn)
			db0, err := util.OpenDB(dsn, c.Concurrency)
			if err != nil {
				log.Fatalf("[cdc-bank] create upstream db client error %v", err)
			}

			if _, err := db0.Exec("set @@tidb_general_log=1"); err != nil {
				log.Errorf("[cdc-bank] enable upstream general log error %v", err)
			}

			// init data
			if _, err := db0.Exec("create table accounts (id int primary key, balance int)"); err != nil {
				log.Fatalf("[cdc-bank] create table error %v", err)
			}
			// split table into regions
			if _, err := db0.Exec(fmt.Sprintf("split table accounts between (%d) AND (%d) REGIONS %d", pk(0), pk(c.Accounts), regions)); err != nil {
				log.Fatalf("[cdc-bank] split table error %v", err)
			}
			for accountID := 0; accountID < c.Accounts; accountID++ {
				if _, err := db0.Exec("insert into accounts values (?, ?)", pk(accountID), initBalance); err != nil {
					log.Fatalf("[cdc-bank] insert account error %v", err)
				}
			}
			upstream = db0
		} else if i == 1 {
			// downstream TiDB
			log.Infof("set up downstream DB connection, dsn: %v", dsn)
			db1, err := util.OpenDB(dsn, validateConcurrency)
			if err != nil {
				log.Fatalf("[cdc-bank] create downstream db client error %v", err)
			}
			if _, err := db1.Exec("set @@tidb_general_log=1"); err != nil {
				log.Errorf("[cdc-bank] enable downstream general log error %v", err)
			}

			// DDL is a strong sync point in TiCDC. Once finishmark table is replicated to downstream
			// all previous DDL and DML are replicated too.
			mustExec(ctx, upstream, `CREATE TABLE IF NOT EXISTS finishmark (foo BIGINT PRIMARY KEY)`)
			waitCtx, waitCancel := context.WithTimeout(ctx, time.Minute)
			waitTable(waitCtx, db1, "finishmark")
			waitCancel()
			log.Info("all tables synced")

			downstream = db1
		}
	}

	var (
		wg      sync.WaitGroup
		count   int32 = 0
		tblChan       = make(chan string, 1)
	)

	// upstream transfers
	for connID := 0; connID < c.Concurrency; connID++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// send ddl periodically
					if atomic.AddInt32(&count, 1)%1000 == 0 {
						tblName := fmt.Sprintf("finishmark%d", atomic.LoadInt32(&count))
						ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (foo BIGINT PRIMARY KEY)", tblName)
						mustExec(ctx, upstream, ddl)

						tblChan <- tblName
						continue
					}

					account1 := rand.Intn(c.Accounts)
					account2 := (rand.Intn(c.Accounts-1) + account1 + 1) % c.Accounts
					txn, err := upstream.Begin()
					if err != nil {
						log.Errorf("[cdc-bank] [connID=%d] begin txn error %v", connID, err)
						continue
					}
					var balance1, balance2 int
					row := txn.QueryRow("select balance from accounts where id = ? for update", pk(account1))
					if err := row.Scan(&balance1); err != nil {
						log.Errorf("[cdc-bank] [connID=%d] select balance1 error %v", connID, err)
						_ = txn.Rollback()
						continue
					}
					row = txn.QueryRow("select balance from accounts where id = ? for update", pk(account2))
					if err := row.Scan(&balance2); err != nil {
						log.Errorf("[cdc-bank] [connID=%d] select balance2 error %v", connID, err)
						_ = txn.Rollback()
						continue
					}
					maxAmount := initBalance / 10
					if maxAmount > balance1 {
						maxAmount = balance1
					}
					amount := rand.Intn(maxAmount)
					newBalance1 := balance1 - amount
					newBalance2 := balance2 + amount

					if _, err := txn.Exec("update accounts set balance = ? where id = ?", newBalance1, pk(account1)); err != nil {
						log.Errorf("[cdc-bank] [connID=%d] update account1 error %v", connID, err)
						_ = txn.Rollback()
						continue
					}
					if _, err := txn.Exec("update accounts set balance = ? where id = ?", newBalance2, pk(account2)); err != nil {
						log.Errorf("[cdc-bank] [connID=%d] update account2 error %v", connID, err)
						_ = txn.Rollback()
						continue
					}
					if err := txn.Commit(); err != nil {
						log.Errorf("[cdc-bank] [connID=%d] commit error %v", connID, err)
						_ = txn.Rollback()
						continue
					}
					log.Infof("[cdc-bank] [connID=%d] transfer %d, %d: %d -> %d, %d: %d -> %d", connID, amount,
						account1, balance1, newBalance1, account2, balance2, newBalance2)
				}
			}
		}(connID)
	}

	var valid, total int64
	// downstream validators
	for validatorIdx := 0; validatorIdx < validateConcurrency; validatorIdx++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for {
				select {
				// DDL is a strong sync point in TiCDC. Once finishmark table is replicated to downstream
				// all previous DDL and DML are replicated too.
				case tblName := <-tblChan:
					waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Minute)
					waitTable(waitCtx, downstream, tblName)
					waitCancel()
					log.Info("ddl synced")

					atomic.AddInt64(&total, 1)

					endTs, err := getDDLEndTs(downstream, tblName)
					if err != nil {
						log.Fatalf("[cdc-bank] get ddl end ts error %v", err)
					}
					// endTs maybe empty due to unknown reason now, if we meet this accidentally, just ignore this round.
					if endTs == "" {
						continue
					}

					atomic.AddInt64(&valid, 1)

					txn, err := downstream.Begin()
					if err != nil {
						log.Errorf("[cdc-bank] [validatorId=%d] begin txn error %v", idx, err)
						continue
					}
					row := txn.QueryRow("select @@tidb_current_ts")
					var startTS int
					if err := row.Scan(&startTS); err != nil {
						log.Errorf("[cdc-bank] [validatorId=%d] select @@tidb_current_ts error %v", idx, err)
						_ = txn.Rollback()
						continue
					}

					if _, err := txn.Exec(fmt.Sprintf("set @@tidb_snapshot='%s'", endTs)); err != nil {
						log.Errorf("[cdc-bank] [validatorId=%d] set @@tidb_snapshot=%s error %v", idx, endTs, err)
						_ = txn.Rollback()
						continue
					}

					row = txn.QueryRow("select sum(balance) from accounts")
					var balanceSum int
					if err := row.Scan(&balanceSum); err != nil {
						log.Errorf("[cdc-bank] [validatorId=%d] select sum(balance) error %v", idx, err)
						_ = txn.Rollback()
						continue
					}

					if balanceSum != expectedSum {
						_ = txn.Rollback()
						log.Fatalf("[cdc-bank] [validatorId=%d] sum: %d, expected: %d, startTS: %d", idx, balanceSum, expectedSum, startTS)
					}

					log.Infof("[cdc-bank] [validatorId=%d] success, startTS: %d, tblName: %s", idx, startTS, tblName)
					if _, err := txn.Exec("set @@tidb_snapshot=''"); err != nil {
						log.Errorf("[cdc-bank][validatorId=%d] set @@tidb_snapshot='' error %v", idx, err)
					}

				case <-ctx.Done():
					return
				default:
					continue
				}
			}
		}(validatorIdx)
	}
	wg.Wait()

	if total == 0 {
		log.Warn("[cdc-bank] finished, but total check round is 0")
	} else {
		log.Infof("[cdc-bank] finished, valid check round: %+v, total try round: %+v, ratio: %f", valid, total, float64(valid)/float64(total))
	}
	return nil
}

func mustExec(ctx context.Context, db *sql.DB, query string) {
	execF := func() error {
		_, err := db.ExecContext(ctx, query)
		return err
	}
	err := util.RunWithRetry(ctx, 3, 100*time.Millisecond, execF)
	if err != nil {
		log.Fatal("exec failed", zap.String("query", query), zap.Error(err))
	}
}

func waitTable(ctx context.Context, db *sql.DB, table string) {
	for {
		if isTableExist(ctx, db, table) {
			return
		}
		log.Info("wait table", zap.String("table", table))
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func isTableExist(ctx context.Context, db *sql.DB, table string) bool {
	// if table is not exist, return true directly
	if _, err := db.Exec("use test"); err != nil {
		log.Fatalf("use db test failed")
	}

	query := fmt.Sprintf("SHOW TABLES LIKE '%s'", table)
	var t string
	err := db.QueryRowContext(ctx, query).Scan(&t)
	if err == nil {
		return true
	}
	if err == sql.ErrNoRows {
		return false
	}

	log.Fatal("query failed", zap.String("query", query), zap.Error(err))
	return false
}

type dataRow struct {
	JobID       int64
	DBName      string
	TblName     string
	JobType     string
	SchemaState string
	SchemeID    int64
	TblID       int64
	RowCount    int64
	StartTime   string
	EndTime     *string
	State       string
}

func getDDLEndTs(db *sql.DB, tableName string) (result string, err error) {
	rows, err := db.Query("admin show ddl jobs")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var line dataRow
	for rows.Next() {
		if err := rows.Scan(&line.JobID, &line.DBName, &line.TblName, &line.JobType, &line.SchemaState, &line.SchemeID,
			&line.TblID, &line.RowCount, &line.StartTime, &line.EndTime, &line.State); err != nil {
			return "", err
		}
		if line.JobType == "create table" && line.TblName == tableName && line.State == "synced" {
			if line.EndTime == nil {
				log.Warnf("ddl end time is null, line=%+v", line)
				return "", nil
			}
			return *line.EndTime, nil
		}
	}
	return "", errors.New(fmt.Sprintf("cannot find in ddl history, tableName: %s", tableName))
}
