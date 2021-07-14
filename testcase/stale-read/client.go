// Copyright 2021 PingCAP, Inc.
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

package staleread

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"sync/atomic"

	"github.com/ngaut/log"
	"github.com/pingcap/tidb/store/tikv/oracle"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

var (
	// Table name
	Table = "stale_read"
)

// ClientCreator creates stale client
type ClientCreator struct {
	Cfg *Config
}

// Config for stale read test
type Config struct {
	DBName          string
	TotalRows       int
	Concurrency     int
	RequestInterval time.Duration
	MaxStaleness    int
}

type staleRead struct {
	*Config
	db      *sql.DB
	counter int64
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Create creates StaleReadClient
func (s *ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &staleRead{
		Config:  s.Cfg,
		counter: 0,
	}
}

// SetUp
func (s *staleRead) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	log.Infof("[stale read] start to init...")

	var err error
	node := clientNodes[idx]
	s.db, err = util.OpenDB(fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, s.DBName), s.Concurrency)
	if err != nil {
		log.Fatal(err)
	}

	// Load data
	s.prepareData()

	return nil
}

func (s *staleRead) encodeTs(t time.Time) uint64 {
	return oracle.ComposeTS(oracle.GetPhysical(t), atomic.AddInt64(&s.counter, 1))
}

// Start
func (s *staleRead) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Info("[stale read] start to test...")
	var wg sync.WaitGroup
	// Write data
	for i := 0; i < s.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(time.Now().Unix()))
			for {
				ids := make([]int, rand.Intn(99)+1)
				for i := range ids {
					ids[i] = rand.Intn(s.TotalRows) + 1
				}
				idsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ids)), ", "), "[]")
				pad := make([]byte, 255)
				util.RandString(pad, rnd)
				stmt := fmt.Sprintf(`UPDATE %s.%s SET pad="%s" WHERE id in (%s)`, s.DBName, Table, pad, idsStr)
				if _, err := s.db.Exec(stmt); err != nil {
					log.Warnf("[stale read] Failed to execute sql [%s], error: %v", stmt, err)
				}
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(s.RequestInterval)
				}
			}
		}()
	}
	// Wait `MaxStaleness` elapsed so all read can reture data
	time.Sleep(time.Duration(s.MaxStaleness) * time.Second)
	// Read data
	for i := 0; i < s.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.mustEqualRead(ctx)
		}()
	}
	wg.Wait()
	return nil
}

func (s *staleRead) mustEqualRead(ctx context.Context) {
	conn1, err := s.db.Conn(ctx)
	if err != nil {
		return
	}
	conn2, err := s.db.Conn(ctx)
	if err != nil {
		return
	}
	count := 0
	for {
		// prepare id
		ids := make([]int, rand.Intn(99)+1)
		for i := range ids {
			ids[i] = rand.Intn(s.TotalRows) + 1
		}
		// 0s-100s ago
		ago := rand.Intn(s.MaxStaleness)
		timeAgo := time.Now().Add(time.Duration(-ago) * time.Second)
		stale, ts, err1 := s.staleRead(ctx, conn1, timeAgo, ids)
		strong, err2 := s.strongRead(ctx, conn2, ts, ids)
		if err1 == nil && err2 == nil {
			// The result should be the same
			if len(stale) != len(strong) || len(strong) == 0 {
				log.Fatalf("got different number of resluts at %d ago, ids: %d, stale: %d, strong: %d, stale res: %v, strong res: %v", ago, len(ids), len(stale), len(strong), stale, strong)
			}
			for id, pad := range strong {
				if stalePad, ok := stale[id]; !ok || stalePad != pad {
					log.Fatalf("got different resluts for id %d at %ds ago, tso: %v, stale read: %s, strong read: %s", id, ago, ts, stale[id], pad)
				}
			}
			if count%10000 == 0 {
				sql := fmt.Sprintf("select * from %s.%s where id in (%s)", s.DBName, Table, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ids)), ", "), "[]"))
				log.Infof("Stale read %d seconds ago equal to strong read, sql: %s", ago, sql)
			}
		} else {
			log.Warnf("read return error, stale: %v, strong: %v", err1, err2)
		}

		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(s.RequestInterval)
			count++
		}
	}
}

func (s *staleRead) staleRead(ctx context.Context, conn *sql.Conn, t time.Time, ids []int) (map[int]string, uint64, error) {
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET TRANSACTION READ ONLY AS OF TIMESTAMP '%s'", t.String())); err != nil {
		log.Warnf("failed to set stale read-only transaction, time: %s, err: %v", t.String(), err)
		return nil, 0, err
	}

	txn, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Warnf("failed to start a read-only transaction, err: %v", err)
		return nil, 0, err
	}
	defer txn.Rollback()

	var tso uint64
	if err = txn.QueryRow("select @@tidb_current_ts").Scan(&tso); err != nil {
		log.Warnf("failed to get tidb_current_ts, err: %v", err)
		return nil, 0, err
	}

	idsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ids)), ", "), "[]")
	rows, err := txn.QueryContext(ctx, fmt.Sprintf("select * from %s.%s where id in (%s)", s.DBName, Table, idsStr))
	if err != nil {
		return nil, tso, err
	}
	defer rows.Close()
	results := make(map[int]string, len(ids))
	for rows.Next() {
		var id int
		var pad string
		if err = rows.Scan(&id, &pad); err != nil {
			return nil, tso, err
		}
		results[id] = pad
	}

	if err := txn.Commit(); err != nil {
		log.Warnf("failed to commit a read-only transaction, err: %v", err)
		return nil, tso, err
	}
	return results, tso, nil
}

func (s *staleRead) strongRead(ctx context.Context, conn *sql.Conn, tso uint64, ids []int) (map[int]string, error) {
	if tso == 0 {
		return nil, errors.New("tso can't be zero")
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("set @@tidb_snapshot = '%d'", tso)); err != nil {
		log.Warnf("failed to set tidb_snapshot, tso: %d, err: %v", tso, err)
		return nil, err
	}

	idsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ids)), ", "), "[]")
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("select * from %s.%s where id in (%s)", s.DBName, Table, idsStr))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make(map[int]string, len(ids))
	for rows.Next() {
		var id int
		var pad string
		if err = rows.Scan(&id, &pad); err != nil {
			return nil, err
		}
		results[id] = pad
	}
	return results, nil
}

func (s *staleRead) prepareData() {
	log.Info("[stale read] prepare data")

	// Create table
	util.MustExec(s.db, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id int(10) PRIMARY KEY, pad varchar(255))", s.DBName, Table))

	// Load data
	var wg sync.WaitGroup
	insertSQL := fmt.Sprintf("INSERT IGNORE INTO %s.%s VALUES", s.DBName, Table)
	for i := 0; i < s.Concurrency; i++ {
		wg.Add(1)
		go func(insertSQL string, item int) {
			defer wg.Done()
			chunk := s.TotalRows / s.Concurrency
			start := item * chunk
			stmt := insertSQL
			unexec := false
			rnd := rand.New(rand.NewSource(time.Now().Unix()))
			for i := 0; i < chunk; i++ {
				pad := make([]byte, 255)
				util.RandString(pad, rnd)
				stmt += fmt.Sprintf(`(%v, "%s")`, strconv.Itoa(start+i+1), pad)
				if i > 0 && i%100 == 0 {
					util.MustExec(s.db, stmt)
					stmt = insertSQL
					unexec = false
				} else if i+1 < chunk {
					stmt += ","
					unexec = true
				}
			}
			if unexec {
				util.MustExec(s.db, stmt)
			}
		}(insertSQL, i)
	}
	wg.Wait()

	log.Info("[stale read] prepare data finish")
	return
}

// TearDown
func (s *staleRead) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}
