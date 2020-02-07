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
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
	"github.com/pingcap/tipocket/pkg/pocket/util"
)

var (
	abTestLogPattern     = regexp.MustCompile(`ab-test-[0-9]+\.log`)
	binlogTestLogPattern = regexp.MustCompile(`single-test-[0-9]+\.log`)
	todoSQLPattern       = regexp.MustCompile(`^\[([0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3} [+-][0-9]{2}:[0-9]{2})\] \[(TODO)\] Exec SQL (.*)$`)
	execIDPattern        = regexp.MustCompile(`^.*?(ab|single)-test-([0-9]+).log$`)
	timeLayout           = `2006/01/02 15:04:05.000 -07:00`
)

func (c *Core) reproduce(ctx context.Context) error {
	var (
		dir   = c.cfg.Options.Path
		table = ""
	)

	if dir == "" {
		log.Fatal("empty dir")
	} else if !util.DirExists(dir) {
		log.Fatal("invalid dir, not exist or not a dir")
	}
	return c.reproduceFromDir(dir, table)
}

func (c *Core) reproduceFromDir(dir, table string) error {
	var logFiles []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		match := false
		if c.cfg.Mode == "abtest" && abTestLogPattern.MatchString(f.Name()) {
			match = true
		} else if c.cfg.Mode == "binlog" && binlogTestLogPattern.MatchString(f.Name()) {
			match = true
		}
		if match {
			logFiles = append(logFiles, path.Join(dir, f.Name()))
		}
	}

	logs, err := c.readLogs(logFiles)
	if err != nil {
		return errors.Trace(err)
	}

	log.Info(len(logs), "logs")
	for index, l := range logs {
		// if index < len(logs) - 1 && logs[index].GetTime() == logs[index + 1].GetTime() {
		// 	log.Info(logs[index].GetNode(), logs[index].GetSQL())
		// 	log.Info(logs[index + 1].GetNode(), logs[index + 1].GetSQL())
		// 	log.Fatal("time mess")
		// }
		if index%100 == 0 {
			log.Info(index, "/", len(logs))
		}
		c.executeByID(l.GetNode(), l.GetSQL())
		// if rand.Float64() < 0.1 {
		// 	ch := make(chan struct{}, 1)
		// 	go e.abTestCompareDataWithoutCommit(ch)
		// 	<- ch
		// }
	}
	log.Info("final check")
	// e.abTestCompareData(false)
	result, err := c.checkConsistency(false)
	if err == nil {
		log.Infof("consistency check %t\n", result)
	}
	return nil
}

func (c *Core) readLogs(logFiles []string) ([]*types.Log, error) {
	var serilizedLogs []*types.Log
	for _, file := range logFiles {
		logs, err := readLogFile(file)
		if err != nil {
			return serilizedLogs, errors.Trace(err)
		}
		serilizedLogs = append(serilizedLogs, logs...)
	}
	sort.Sort(types.ByLog(serilizedLogs))
	return serilizedLogs, nil
}

func readLogFile(logFile string) ([]*types.Log, error) {
	var (
		execID = parseExecNumber(logFile)
		logs   []*types.Log
	)
	f, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("error when open file %v\n", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("error when close file %v", err)
		}
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		log, err := parseLog(line, execID)
		if err == nil {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

func parseExecNumber(filePath string) int {
	m := execIDPattern.FindStringSubmatch(filePath)
	if len(m) != 3 {
		return 0
	}
	id, err := strconv.Atoi(m[2])
	if err != nil {
		return 0
	}
	return id
}

func parseLog(line string, node int) (*types.Log, error) {
	var (
		m   []string
		log types.Log
	)

	m = todoSQLPattern.FindStringSubmatch(line)
	if len(m) != 4 {
		return nil, errors.NotFoundf("not matched line %s", line)
	}

	t, err := time.Parse(timeLayout, m[1])
	if err != nil {
		return nil, err
	}
	log.Time = t
	log.SQL = &types.SQL{
		SQLType: parseSQLType(m[3]),
		SQLStmt: m[3],
	}
	log.State = m[2]

	log.Node = node
	return &log, nil
}

func parseSQLType(sql string) types.SQLType {
	sql = strings.ToLower(sql)
	if strings.HasPrefix(sql, "select") {
		return types.SQLTypeDMLSelect
	}
	if strings.HasPrefix(sql, "update") {
		return types.SQLTypeDMLUpdate
	}
	if strings.HasPrefix(sql, "insert") {
		return types.SQLTypeDMLInsert
	}
	if strings.HasPrefix(sql, "delete") {
		return types.SQLTypeDMLDelete
	}
	if strings.HasPrefix(sql, "create table") {
		return types.SQLTypeDDLCreateTable
	}
	if strings.HasPrefix(sql, "alter table") {
		return types.SQLTypeDDLAlterTable
	}
	if strings.HasPrefix(sql, "create index") {
		return types.SQLTypeDDLCreateIndex
	}
	if strings.HasPrefix(sql, "begin") {
		return types.SQLTypeTxnBegin
	}
	if strings.HasPrefix(sql, "commit") {
		return types.SQLTypeTxnCommit
	}
	if strings.HasPrefix(sql, "rollback") {
		return types.SQLTypeTxnRollback
	}
	return types.SQLTypeUnknown
}
