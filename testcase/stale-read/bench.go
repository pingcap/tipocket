package staleread

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/pingcap/errors"
	"github.com/tiancaiamao/sysbench"
	"math/rand"
)

const createTableTemplate = `create table if not exists sbtest%d (
id int(11) not null primary key,
k int(11) not null,
c char(120) not null default '',
pad char(255) not null default '')`

const splitTableTemplate = `SPLIT TABLE sbtest%d BETWEEN (0) AND (1000000000) REGIONS 100;`

type SysbenchCase struct {
	insertCount    int
	rowsEachInsert int
	dbHost         string
}

func (c *SysbenchCase) CreateTable(db *sql.DB) error {
	db.Exec(fmt.Sprintf(createTableTemplate, 0))
	db.Exec(fmt.Sprintf(splitTableTemplate, 0))
	return nil
}

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
			return errors.WithStack(err)
		}
	}
	return nil
}

func (c *SysbenchCase) Execute(worker *sysbench.Worker, db *sql.DB) error {
	for i := 0; i < 100; i++ {
		err := c.executeSET(db)
		if err != nil {
			return err
		}
		err = c.executeSTART(db)
		if err != nil {
			return err
		}
		err = c.executeSelect(db)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *SysbenchCase) executeSET(db *sql.DB) error {
	num := c.insertCount * c.rowsEachInsert
	_, err := db.Exec("SET TRANSACTION READ ONLY AS OF TIMESTAMP tidb_bounded_staleness(DATE_SUB(NOW(), INTERVAL 10 SECOND)")
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

func (c *SysbenchCase) executeSTART(db *sql.DB) error {
	num := c.insertCount * c.rowsEachInsert
	_, err := db.Exec("START TRANSACTION READ ONLY AS OF TIMESTAMP tidb_bounded_staleness(DATE_SUB(NOW(), INTERVAL 10 SECOND)")
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
	rows, err := db.Query("select id, k, c, pad from sbtest0 as of timestamp tidb_bounded_staleness(DATE_SUB(NOW(), INTERVAL 10 SECOND) where k in (?, ?, ?)", rand.Intn(num), rand.Intn(num), rand.Intn(num))
	defer rows.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

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
