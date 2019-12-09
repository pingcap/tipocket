package core

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	smith "github.com/pingcap/tipocket/go-sqlsmith"
	"github.com/pingcap/tipocket/abclient/pkg/types"
	"github.com/pingcap/tipocket/abclient/executor"
	"github.com/pingcap/tipocket/abclient/util"
)

func (e *Executor) reloadSchema() error {
	schema, err := e.coreConn.FetchSchema(e.dbname)
	if err != nil {
		return errors.Trace(err)
	}
	e.ss = smith.New()
	e.ss.LoadSchema(schema)
	e.ss.SetDB(e.dbname)
	e.ss.SetStable(e.coreOpt.Stable)
	return nil
}

func (e *Executor) startDataCompare() {
	switch e.mode {
	case "abtest":
		e.startABTestDataCompare()
	case "binlog":
		e.startBinlogTestDataCompare()
	default:
		log.Fatalf("unhandled mode %s", e.mode)
	}
}

func (e *Executor) startABTestDataCompare() {
	c := time.Tick(time.Minute)
	time := 0
	for range c {
		if e.waitingForCheck {
			continue
		}
		e.waitingForCheck = true
		time++
		go func(time int) {
			log.Info("ready to compare data")
			result, err := e.abTestCompareData(true)
			log.Infof("test %d compare data result %t\n", time, result)
			if err != nil {
				log.Fatalf("compare data error %+v", errors.ErrorStack(err))
			}
			e.waitingForCheck = false
		}(time)
	}
}

func (e *Executor) startBinlogTestDataCompare() {
	c := time.Tick(time.Minute)
	time := 0
	for range c {
		if e.waitingForCheck {
			continue
		}
		e.waitingForCheck = true
		time++
		go func(time int) {
			log.Info("ready to compare binlog data")
			result, err := e.binlogTestCompareData(true)
			log.Infof("test %d compare binlog data result %t\n", time, result)
			if err != nil {
				log.Fatalf("compare binlog data error %+v", errors.ErrorStack(err))
			}
			e.waitingForCheck = false
		}(time)
	}
}

func (e *Executor) checkConsistency(delay bool) (bool, error) {
	switch e.mode {
	case "abtest":
		return e.abTestCompareData(delay)
	case "binlog":
		return e.binlogTestCompareData(delay)
	}
	// panic here?
	return true, nil
}

// abTestCompareDataWithoutCommit take snapshot without other transactions all committed
// this function can run async and channel is for waiting taking snapshot complete
func (e *Executor) abTestCompareDataWithoutCommit(ch chan struct{}) {
	// start a temp session for keep the snapshot of state
	opt := e.execOpt.Clone()
	opt.Mute = true
	compareExecutor, err := executor.NewABTest(e.dsn1, e.dsn2, opt)
	if err != nil {
		log.Fatal(err)
	}
	// schema should be fetch first
	schema, err := compareExecutor.GetConn().FetchSchema(e.dbname)
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
	
	time.Sleep(time.Duration(rand.Intn(1000))*time.Millisecond)
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

func (e *Executor) abTestCompareData(delay bool) (bool, error) {
	// only for abtest
	if e.mode != "abtest" {
		return true, nil
	}

	// start a temp session for keep the snapshot of state
	compareExecutor, err := executor.NewABTest(e.dsn1, e.dsn2, e.execOpt.Clone())
	if err != nil {
		return false, errors.Trace(err)
	}
	defer func(compareExecutor *executor.Executor) {
		if err := compareExecutor.Close(); err != nil {
			log.Fatal("close compare executor error %+v\n", errors.ErrorStack(err))
		}
	}(compareExecutor)

	// commit or rollback all transactions
	e.Lock()
	// no async here to ensure all transactions are committed or rollbacked in order
	// use resolveDeadLock func to avoid deadlock
	e.resolveDeadLock()
	schema, err := compareExecutor.GetConn().FetchSchema(e.dbname)
	if err != nil {
		e.Unlock()
		return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestTxnBegin(); err != nil {
		e.Unlock()
		return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestSelect(makeCompareSQLs(schema)[0]); err != nil {
		log.Fatal(err)
	}
	// free the lock since the compare has already got the same snapshot in both side
	// go on other transactions
	// defer here for protect environment
	defer e.Unlock()

	// delay will hold on this snapshot and check it later
	if delay {
		time.Sleep(time.Duration(rand.Intn(5))*time.Second)
	}
	
	return e.compareData(compareExecutor, schema)
}

func (e *Executor) binlogTestCompareData(delay bool) (bool, error) {
	// only for binlog test
	if e.mode != "binlog" {
		return true, nil
	}

	// start a temp session for keep the snapshot of state
	compareExecutor, err := executor.NewABTest(e.dsn1, e.dsn2, e.execOpt.Clone())
	if err != nil {
		return false, errors.Trace(err)
	}
	defer func(compareExecutor *executor.Executor) {
		if err := compareExecutor.Close(); err != nil {
			log.Fatal("close compare executor error %+v\n", errors.ErrorStack(err))
		}
	}(compareExecutor)

	// commit or rollback all transactions
	// lock here before get snapshot
	e.Lock()
	// no async here to ensure all transactions are committed or rollbacked in order
	// use resolveDeadLock func to avoid deadlock
	e.resolveDeadLock()

	// insert a table and wait for the sync job is done
	table, tableStmt := generateWaitTable()
	compareExecutor.SingleTestCreateTable(tableStmt)
	syncDone := false
	for !syncDone {
		time.Sleep(time.Second)
		tables, err := compareExecutor.GetConn2().FetchTables(e.dbname)
		if err != nil {
			return false, errors.Trace(err)
		}
		for _, t := range tables {
			if t == table {
				syncDone = true
			}
		}
		log.Info("got sync status", syncDone)
	}

	schema, err := compareExecutor.GetConn().FetchSchema(e.dbname)
	if err != nil {
		e.Unlock()
		return false, errors.Trace(err)
	}
	if err := compareExecutor.ABTestTxnBegin(); err != nil {
		e.Unlock()
		return false, errors.Trace(err)
	}
	log.Info("compare wait for chan finish")
	// free the lock since the compare has already got the same snapshot in both side
	// go on other transactions
	// defer here for protect environment
	defer e.Unlock()

	// delay will hold on this snapshot and check it later
	if delay {
		time.Sleep(time.Duration(rand.Intn(5))*time.Second)
	}
	
	return e.compareData(compareExecutor, schema)
}

func (e *Executor) compareData(beginnedConnect *executor.Executor, schema [][5]string) (bool, error) {
	sqls := makeCompareSQLs(schema)
	for _, sql := range sqls {
		if err := beginnedConnect.ABTestSelect(sql); err != nil {
			log.Fatalf("inconsistency when exec %s compare data %+v, begin: %s\n",
				sql, err, util.FormatTimeStrAsLog(beginnedConnect.GetConn().GetBeginTime()))
		}
	}
	log.Info("consistency check pass")
	return true, nil
}

func makeCompareSQLs (schema [][5]string) []string {
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

func generateWaitTable () (string, string) {
	sec := time.Now().Unix()
	table := fmt.Sprintf("t%d", sec)
	return table, fmt.Sprintf("CREATE TABLE %s(id int)", table)
}
