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

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config is for sqllogicClient
type Config struct {
	SkipError           bool
	TaskCount           int
	CaseURL             string
	TestDir             string
	ReplicaRead         string
	DbName              string
	TiFlashDataReplicas int
}

type sqllogicClient struct {
	*Config
	ip   string
	port int32
}

// ClientCreator creates sqllogicClient
type ClientCreator struct {
	*Config
}

// Create creates sqllogicClient
func (l *ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &sqllogicClient{
		Config: l.Config,
	}
}

// SetUp set up sqllogicClient
func (c *sqllogicClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	node := clientNodes[idx]
	c.ip, c.port = node.IP, node.Port
	dbDSN := fmt.Sprintf("root:@tcp(%s:%d)/%s", node.IP, node.Port, c.DbName)
	db, err := util.OpenDB(dbDSN, 1)
	if err != nil {
		return err
	}
	defer db.Close()

	for index := 0; index < c.TaskCount; index++ {
		dropSQL := fmt.Sprintf("drop database if exists sqllogic_test_%d;", index)
		err := util.RunWithRetry(ctx, dbTryNumber, 3, func() error {
			_, err := db.Exec(dropSQL)
			return err
		})
		if err != nil {
			return errors.Errorf("executing %s err %v", dropSQL, err)
		}

		createSQL := fmt.Sprintf("create database if not exists sqllogic_test_%d;", index)
		err = util.RunWithRetry(ctx, dbTryNumber, 3*time.Second, func() error {
			_, err := db.Exec(createSQL)
			return err
		})
		if err != nil {
			return errors.Errorf("executing %s err %v", createSQL, err)
		}
	}

	if err := util.InstallArchive(ctx, c.CaseURL, "./"); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// TearDown does nothing
func (c *sqllogicClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start starts test
func (c *sqllogicClient) Start(ctx context.Context, _ interface{}, clientNodes []cluster.ClientNode) error {
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
	dbs := createDatabases(c.TaskCount, fmt.Sprintf("%s:%d", c.ip, c.port), "root", "", c.ReplicaRead)
	defer closeDatabases(dbs)

	go addTasks(ctx, fileNames, taskChan)

	for i := 0; i < c.TaskCount; i++ {
		go doProcess(ctx, doneChan, taskChan, resultChan, dbs[i], i, c.TiFlashDataReplicas, c.SkipError)
	}

	go doWait(ctx, doneChan, resultChan, c.TaskCount)

	if errCnt, errMsg := doResult(ctx, resultChan, startTime); errCnt > 0 {
		log.Fatalf("test failed, error count:%d, error message%s", errCnt, errMsg)
	}
	return nil
}
