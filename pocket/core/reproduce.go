package core

import (
	"bufio"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"strconv"
	"time"
	"sort"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/util"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

var (
	abTestLogPattern = regexp.MustCompile(`ab-test-[0-9]+\.log`)
	binlogTestLogPattern = regexp.MustCompile(`single-test-[0-9]+\.log`)
	todoSQLPattern = regexp.MustCompile(`^\[([0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3} [+-][0-9]{2}:[0-9]{2})\] \[(TODO)\] Exec SQL (.*)$`)
	execIDPattern = regexp.MustCompile(`^.*?(ab|single)-test-([0-9]+).log$`)
	timeLayout = `2006/01/02 15:04:05.000 -07:00`
)

func (e *Executor) reproduce() {
	reproduceParams := strings.Split(e.coreOpt.Reproduce, ":")
	var (
		dir string
		table string
	)

	if len(reproduceParams) >= 1 {
		dir = reproduceParams[0]
	}
	if len(reproduceParams) >= 2 {
		table = reproduceParams[1]
	}

	if dir == "" {
		log.Fatal("empty dir")
	} else if !util.DirExists(dir) {
		log.Fatal("invalid dir, not exist or not a dir")
	}
	e.reproduceFromDir(dir, table)
}

func (e *Executor) reproduceFromDir(dir, table string) {
	var logFiles []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		match := false
		if e.mode == "abtest" && abTestLogPattern.MatchString(f.Name()) {
			match = true
		} else if e.mode == "binlog" && binlogTestLogPattern .MatchString(f.Name()) {
			match = true
		}
		if match {
			logFiles = append(logFiles, path.Join(dir, f.Name()))
		}
	}

	logs := e.readLogs(logFiles)

	for index, l := range logs {
		if index < len(logs) - 1 && logs[index].GetTime() == logs[index + 1].GetTime() {
			log.Info(logs[index].GetNode(), logs[index].GetSQL())
			log.Info(logs[index + 1].GetNode(), logs[index + 1].GetSQL())
			log.Fatal("time mess")
		}
		e.ExecStraight(l.GetSQL(), l.GetNode())
		// if rand.Float64() < 0.1 {	
		// 	ch := make(chan struct{}, 1)
		// 	go e.abTestCompareDataWithoutCommit(ch)
		// 	<- ch
		// }
	}
	log.Info("final check")
	// e.abTestCompareData(false)
	result, err := e.checkConsistency(false)
	if err == nil {
		log.Infof("consistency check %t\n", result)
	}
	os.Exit(0)
}

func (e *Executor) readLogs (logFiles []string) []*types.Log {
	var serilizedLogs []*types.Log
	for _, file := range logFiles {
		logs, err := e.readLogFile(file)
		if err == nil {
			serilizedLogs = append(serilizedLogs, logs...)
		}
	}
	sort.Sort(types.ByLog(serilizedLogs))
	return serilizedLogs
}

func (e *Executor) readLogFile(logFile string) ([]*types.Log, error) {
	var (
		execID = parseExecNumber(logFile)
		logs []*types.Log
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
		m []string
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
	if strings.HasPrefix(sql, "create") {
		return types.SQLTypeDDLCreate
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
