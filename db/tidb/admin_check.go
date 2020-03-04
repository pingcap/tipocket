package tidb

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"

	"k8s.io/apimachinery/pkg/util/wait"
)

// AdminChecker returns a adminChecker instance
func AdminChecker(cfg *control.Config) core.Checker {
	return adminChecker{cfg: cfg}
}

const (
	tiDBGCTime = 720 * time.Hour
)

type adminChecker struct {
	cfg *control.Config
}

func (ac adminChecker) Check(_ core.Model, _ []core.Operation) (bool, error) {
	if len(ac.cfg.ClientNodes) == 0 {
		return false, errors.New("client node is empty")
	}

	index := rand.Intn(len(ac.cfg.ClientNodes))
	node := ac.cfg.ClientNodes[index]
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port))
	if err != nil {
		return false, nil
	}

	if err := adjustTiDBGC(db); err != nil {
		return false, err
	}

	if err := checkAction(db); err != nil {
		return false, err
	}
	return true, nil
}

func (adminChecker) Name() string {
	return "tidb_admin_checker"
}

func adjustTiDBGC(db *sql.DB) error {
	err := wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		stmt := fmt.Sprintf("update mysql.tidb set variable_value='%s' where variable_name='tikv_gc_life_time'; "+
			"update mysql.tidb set variable_value='%s' where variable_name='tikv_gc_run_interval';", tiDBGCTime.String(), tiDBGCTime.String())
		_, err := db.Exec(stmt)
		if err != nil {
			log.Errorf("failed to adjust tidb gc time, error: %v", err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkAction(db *sql.DB) error {
	databases, err := getAllDatabases(db)
	if err != nil {
		log.Errorf("get databases failed: %v", err)
		return errors.Trace(err)
	}

	for _, dbName := range databases {
		if err = checkDatabase(db, dbName); err != nil {
			log.Errorf("check database %s failed: %v", dbName, err)
			continue
		}
	}

	return nil
}

func checkDatabase(db *sql.DB, dbName string) error {
	useStmt := fmt.Sprintf("USE %s", dbName)
	if _, err := db.Exec(useStmt); err != nil {
		log.Errorf("exec %s failed: %v", useStmt, err)
		return errors.Trace(err)
	}

	err := addTable(db)
	if err != nil {
		log.Errorf("add table failed %v", err)
		return errors.Trace(err)
	}

	tables, err := getAllTables(db)
	if err != nil {
		log.Errorf("databases %s get tables failed: %v", dbName, err)
		return errors.Trace(err)
	}

	for _, table := range tables {
		log.Infof("start to check %s.%s", dbName, table)
		if err = checkTable(db, dbName, table); err != nil {
			log.Errorf(" check %s.%s failed: %v", dbName, table, err)
			continue
		}
	}

	return nil
}

func checkTable(db *sql.DB, dbName, table string) error {
	stmt := fmt.Sprintf("admin check table `%s`", table)
	log.Infof("start to exec %s", stmt)
	_, err := db.Exec(stmt)
	if err != nil && strings.Contains(err.Error(), "1105") && !getFilter(err) {
		return fmt.Errorf("check %s.%s data nonconsistent %v", dbName, table, err)
	}

	return errors.Trace(err)
}

func getAllDatabases(db *sql.DB) ([]string, error) {
	databases := []string{}
	stmt := "SHOW DATABASES"
	rows, err := db.Query(stmt)
	if err != nil {
		log.Errorf("exec %s failed: %v", stmt, err)
		return databases, errors.Trace(err)
	}

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			log.Errorf("scan dbname failed: %v", err)
			return databases, errors.Trace(err)
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

func getAllTables(db *sql.DB) ([]string, error) {
	tables := []string{}
	stmt := "SHOW TABLES"
	rows, err := db.Query(stmt)
	if err != nil {
		log.Errorf("exec %s failed: %v", stmt, err)
		return tables, errors.Trace(err)
	}

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			log.Errorf("scan table failed: %v", err)
			return tables, errors.Trace(err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func addTable(db *sql.DB) error {
	stmt := "drop table if exists check_table_t1;CREATE TABLE check_table_t1 (c2 BOOL, PRIMARY KEY (c2));INSERT INTO check_table_t1 SET c2 = '0';ALTER TABLE check_table_t1 ADD COLUMN c3 DATETIME NULL DEFAULT '2668-02-03 17:19:31';ALTER TABLE check_table_t1 ADD INDEX idx2 (c3);"
	log.Infof(" start to add table %s", stmt)
	_, err := db.Exec(stmt)
	if err != nil && strings.Contains(err.Error(), "1105") && !getFilter(err) {
		return fmt.Errorf("add table check_table_t1 data nonconsistent %v", err)
	}

	return errors.Trace(err)
}

func getFilter(err error) bool {
	return strings.Contains(err.Error(), "cancelled DDL job") || strings.Contains(err.Error(), "Information schema is changed") ||
		strings.Contains(err.Error(), "TiKV server timeout") || strings.Contains(err.Error(), "Error 9007: Write conflict")
}
