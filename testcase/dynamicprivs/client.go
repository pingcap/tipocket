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

package dynamicprivs

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/util"
)

// CaseCreator is a creator of test client
type CaseCreator struct{}

// Create creates a test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	return &Client{}
}

// Client defines how our test case works
type Client struct {
	db      *sql.DB
	tempdir string
}

// SetUp implements the core.Client interface.
func (c *Client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	log.Info("start to setup client...")
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
	util.SetMySQLProxy(fixture.Context.MySQLProxy)
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("open db error: %v", err)
	}

	util.MustExec(db, "drop table if exists t")
	util.MustExec(db, "create table t(id int)")
	util.MustExec(db, "insert into t(id) values(1)")

	// Create a user that only has the privileges to backup
	// and nothing else. Make sure the test confirms these
	// are the minimal privileges.

	util.MustExec(db, "DROP USER IF EXISTS backupadmin")
	util.MustExec(db, "CREATE USER backupadmin")
	util.MustExec(db, "GRANT BACKUP_ADMIN, RESTORE_ADMIN ON *.* TO backupadmin")

	if err := db.Close(); err != nil {
		log.Fatalf("could not close initial connection due to: %v", err)
	}

	dsn = fmt.Sprintf("backupadmin@tcp(%s:%d)/information_schema", node.IP, node.Port)
	db, err = util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("open db error: %v", err)
	}

	// Create a tempdir.
	c.tempdir, err = ioutil.TempDir("", "dynamicprivstest")
	if err != nil {
		log.Fatalf("failed to setup temp dir: %v", err)
	}

	c.db = db
	return nil
}

// TearDown implements the core.Client interface.
func (c *Client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	// delete the tempdir.
	os.RemoveAll(c.tempdir)
	return nil
}

// Start implements the core.StandardClientExtensions interface.
func (c *Client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {

	expectedLocation := fmt.Sprintf("local://%s", c.tempdir)
	backupStmt := fmt.Sprintf("BACKUP DATABASE * TO '%s'", expectedLocation)
	log.Info("creating a backup in: " + c.tempdir)
	rows, err := c.db.QueryContext(ctx, backupStmt)
	if err != nil {
		return err
	}
	var (
		location      string
		size          string
		backupTs      string
		queueTime     string
		executionTime string
	)
	if rows.Next() {
		if err := rows.Scan(&location, &size, &backupTs, &queueTime, &executionTime); err != nil {
			return err
		}
	}
	if location != expectedLocation {
		return fmt.Errorf("expected backup to %s, but got %s", expectedLocation, location)
	}
	rows.Close()
	log.Info("backup complete")

	// In future we should try restoring as well, but it currently won't work due to
	// https://github.com/pingcap/tidb/issues/24912
	restoreStmt := fmt.Sprintf("RESTORE DATABASE * FROM '%s'", expectedLocation)
	log.Info("restoring database from: ", expectedLocation)
	rows, err = c.db.QueryContext(ctx, restoreStmt)
	if err != nil {
		return err
	}
	defer rows.Close()
	log.Info("restore completed")
	return nil
}
