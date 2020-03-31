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

package sqllogictest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config is for sqllogicClient
type Config struct {
	SkipError bool
	TaskCount int
	CaseURL   string
	TestDir   string
}

type sqllogicClient struct {
	*Config
	ip   string
	port int32
}

// CaseCreator creates sqllogicClient
type CaseCreator struct {
	*Config
}

// Create creates sqllogicClient
func (l *CaseCreator) Create(node types.ClientNode) core.Client {
	return &sqllogicClient{
		Config: l.Config,
	}
}

// SetUp set up sqllogicClient
func (c *sqllogicClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	node := nodes[idx]
	c.ip, c.port = node.IP, node.Port
	dbDSN := fmt.Sprintf("root:@tcp(%s:%d)/test", node.IP, node.Port)
	db, err := util.OpenDB(dbDSN, 1)
	if err != nil {
		return err
	}
	defer db.Close()

	for index := 0; index < c.TaskCount; index++ {
		dropSql := fmt.Sprintf("drop database if exists sqllogic_test_%d;", index)
		err := util.RunWithRetry(ctx, dbTryNumber, 3, func() error {
			_, err := db.Exec(dropSql)
			return err
		})
		if err != nil {
			return errors.Errorf("executing %s err %v", dropSql, err)
		}

		createSql := fmt.Sprintf("create database if not exists sqllogic_test_%d;", index)
		err = util.RunWithRetry(ctx, dbTryNumber, 3*time.Second, func() error {
			_, err := db.Exec(createSql)
			return err
		})
		if err != nil {
			return errors.Errorf("executing %s err %v", createSql, err)
		}
	}

	if err := util.InstallArchive(ctx, c.CaseURL, "./"); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// TearDown does nothing
func (c *sqllogicClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

// Invoke does nothing
func (c *sqllogicClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

// NextRequest does nothing
func (c *sqllogicClient) NextRequest() interface{} {
	panic("implement me")
}

// DumpState does nothing
func (c *sqllogicClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

// Start starts test
func (c *sqllogicClient) Start(ctx context.Context, _ interface{}, clientNodes []types.ClientNode) error {
	startTime := time.Now()
	var fileNames []string
	filepath.Walk(c.TestDir, func(testPath string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(testPath, ".test") {
			return nil
		}

		fileNames = append(fileNames, testPath)
		return nil
	})

	if c.TaskCount > len(fileNames) {
		c.TaskCount = len(fileNames)
	}
	if c.TaskCount <= 0 {
		return nil
	}
	taskChan := make(chan string, c.TaskCount)
	doneChan := make(chan struct{}, c.TaskCount)
	resultChan := make(chan *result, c.TaskCount)
	dbs := createDatabases(c.TaskCount, fmt.Sprintf("%s:%d", c.ip, c.port), "root", "")
	defer closeDatabases(dbs)

	go addTasks(ctx, fileNames, taskChan)

	for i := 0; i < c.TaskCount; i++ {
		go doProcess(ctx, doneChan, taskChan, resultChan, dbs[i], i, c.SkipError)
	}

	go doWait(ctx, doneChan, resultChan, c.TaskCount)

	if errCnt, errMsg := doResult(ctx, resultChan, startTime); errCnt > 0 {
		log.Fatalf("test failed, error count:%d, error message%s", errCnt, errMsg)
	}
	return nil
}
