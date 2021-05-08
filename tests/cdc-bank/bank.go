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
	"time"

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
			// wait at most 60 seconds for initialization of downstream
			for j := 0; j < 60; j++ {
				var sumBalance int
				row := db1.QueryRow("select sum(balance) from accounts")
				err := row.Scan(&sumBalance)
				if err != nil || sumBalance != expectedSum {
					log.Infof("wait until downstream is initialized")
					time.Sleep(time.Second)
					continue
				}
				break
			}
			downstream = db1
		}
	}

	var wg sync.WaitGroup
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

	// downstream validators
	for validatorIdx := 0; validatorIdx < validateConcurrency; validatorIdx++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
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
					row = txn.QueryRow("select sum(balance) from accounts")
					var balanceSum int
					if err := row.Scan(&balanceSum); err != nil {
						log.Errorf("[cdc-bank] [validatorId=%d] select sum(balance) error %v", idx, err)
						_ = txn.Rollback()
						continue
					}
					if balanceSum == expectedSum {
						_ = txn.Rollback()
						log.Infof("[cdc-bank] [validatorId=%d] success, startTS: %d", idx, startTS)
						return
					}

					log.Errorf("[cdc-bank] [validatorId=%d] sum: %d, expected: %d, startTS: %d", idx, balanceSum, expectedSum, startTS)
				}
			}
		}(validatorIdx)
	}
	wg.Wait()
	return nil
}
