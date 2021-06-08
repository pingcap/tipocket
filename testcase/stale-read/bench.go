package staleread

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/tiancaiamao/sysbench"
	"go.uber.org/zap"
)

const createTableTemplate = `create table if not exists sbtest%d (
id int(11) not null primary key,
k int(11) not null,
c char(120) not null default '',
pad char(255) not null default '')`

const splitTableTemplate = `SPLIT TABLE sbtest%d BETWEEN (0) AND (1000000000) REGIONS 100;`

// SysbenchCase indicates a sysbench case
type SysbenchCase struct {
	insertCount    int
	rowsEachInsert int
}

// CreateTable ...
func (c *SysbenchCase) CreateTable(db *sql.DB) error {
	if err := c.DropTable(db); err != nil {
		log.Error("fail to drop table", zap.Error(err))
		return err
	}
	if _, err := db.Exec(fmt.Sprintf(createTableTemplate, 0)); err != nil {
		log.Error("fail to create table", zap.Error(err))
		return err
	}
	if _, err := db.Exec(fmt.Sprintf(splitTableTemplate, 0)); err != nil {
		log.Error("fail to split table", zap.Error(err))
		return err
	}
	return nil
}

// InsertData ...
func (c *SysbenchCase) InsertData(worker *sysbench.Worker, db *sql.DB) error {
	var buf bytes.Buffer
	pkID := worker.ID
	for i := 0; i < c.insertCount; i++ {
		buf.Reset()
		fmt.Fprintf(&buf, "insert into sbtest%d (id, k, c, pad) values ", 0)
		for i := 0; i < c.rowsEachInsert; i++ {
			pkID = nextPrimaryID(worker.Count, pkID)
			dot := ""
			if i > 0 {
				dot = ", "
			}
			fmt.Fprintf(&buf, "%s(%d, %d, '%s', '%s')", dot, pkID, rand.Intn(1<<11), randString(32), randString(32))
		}

		_, err := db.Exec(buf.String())
		if err != nil {
			log.Info("Insert data error", zap.Error(err))
			return errors.WithStack(err)
		}
	}
	log.Info("insert data finish")
	return nil
}

// Execute ...
// TODO: fulfill workload in future
func (c *SysbenchCase) Execute(worker *sysbench.Worker, db *sql.DB) error {
	log.Info("worker start execute")
	err := c.executeSET(db)
	if err != nil {
		log.Info("execute set transaction read only as of testcase fail", zap.Error(err))
		return err
	}
	err = c.executeSelect(db)
	if err != nil {
		log.Info("execute select as of timestamp fail", zap.Error(err))
		return err
	}
	log.Info("worker start success")
	return nil
}

func (c *SysbenchCase) executeSET(db *sql.DB) error {
	num := c.insertCount * c.rowsEachInsert
	now := time.Now()
	previous := now.Add(-3 * time.Second)
	nowStr := now.Format("2006-1-2 15:04:05.000")
	previousStr := previous.Format("2006-1-2 15:04:05.000")
	setSQL := fmt.Sprintf(`SET TRANSACTION READ ONLY as of timestamp tidb_bounded_staleness('%v', '%v')`, previousStr, nowStr)
	_, err := db.Exec(setSQL)
	if err != nil {
		return err
	}
	rows, err := db.Query("select id, k, c, pad from sbtest0 where k in (?, ?, ?)", rand.Intn(num), rand.Intn(num), rand.Intn(num))
	defer rows.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// TODO: don't know why this case failed
func (c *SysbenchCase) executeSTART(db *sql.DB) error {
	num := c.insertCount * c.rowsEachInsert
	now := time.Now()
	previous := now.Add(-3 * time.Second)
	nowStr := now.Format("2006-1-2 15:04:05.000")
	previousStr := previous.Format("2006-1-2 15:04:05.000")
	startSQL := fmt.Sprintf(`START TRANSACTION READ ONLY AS OF TIMESTAMP tidb_bounded_staleness('%v', '%v')`, previousStr, nowStr)
	_, err := db.Exec(startSQL)
	if err != nil {
		return err
	}
	rows, err := db.Query("select id, k, c, pad from sbtest0 where k in (?, ?, ?)", rand.Intn(num), rand.Intn(num), rand.Intn(num))
	defer rows.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = db.Exec("COMMIT")
	if err != nil {
		return err
	}
	return nil
}

func (c *SysbenchCase) executeSelect(db *sql.DB) error {
	num := c.insertCount * c.rowsEachInsert
	now := time.Now()
	previous := now.Add(-3 * time.Second)
	nowStr := now.Format("2006-1-2 15:04:05.000")
	previousStr := previous.Format("2006-1-2 15:04:05.000")
	selectSQL := fmt.Sprintf("select id, k, c, pad from sbtest0 as of timestamp tidb_bounded_staleness('%v','%v') where k in (%v, %v, %v)", previousStr, nowStr, rand.Intn(num), rand.Intn(num), rand.Intn(num))
	rows, err := db.Query(selectSQL)
	defer rows.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// DropTable ...
func (c *SysbenchCase) DropTable(db *sql.DB) error {
	_, err := db.Exec("drop table if exists sbtest0")
	return err
}

const ascii = "abcdefghijklmnopqrstuvwxyz1234567890"

func randString(n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		pos := rand.Intn(len(ascii))
		buf.WriteByte(ascii[pos])
	}
	return buf.String()
}

func nextPrimaryID(workerCount int, current int) int {
	return current + workerCount
}
