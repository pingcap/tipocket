package pkg

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngaut/log"
	"golang.org/x/net/context"
)

// Client is for pessimistic transaction test
type Client struct {
	cfg        ClientConfig
	successTxn uint64
	failTxn    uint64
}

// ClientConfig is for pessimistic test case.
type ClientConfig struct {
	DBName         string `toml:"dbname"`
	Concurrency    int    `toml:"concurrency"`
	TableNum       int    `toml:"table_num"`
	TableSize      uint64 `toml:"table_size"`
	OperationCount uint64 `toml:"operation_count"`
	Mode           string `toml:"mode"`
	InsertDelete   bool   `toml:"insert_delete"`
	IgnoreCodesO   []int  `toml:"ignore_codes_o"`
	IgnoreCodesP   []int  `toml:"ignore_codes_p"`
	UsePrepareStmt bool   `toml:"use_prepare_stmt"`
}

const (
	batchSize       = 100
	tableNamePrefix = "t"
	caseName        = "pessimistic transaction"
)

// NewPessimisticCase ...
func NewPessimisticCase(cfg ClientConfig) *Client {
	return &Client{cfg: cfg}
}

// Initialize ...
func (c *Client) Initialize(ctx context.Context, db *sql.DB) error {
	for i := 0; i < c.cfg.TableNum; i++ {
		tableName := fmt.Sprintf("%s%d", tableNamePrefix, i)
		log.Infof("preparing table: %s\n", tableName)
		if _, err := db.Exec(fmt.Sprintf("drop table if exists %s", tableName)); err != nil {
			return nil
		}
		createTableStmt := fmt.Sprintf("create table %s ("+
			"id bigint primary key,"+
			"u bigint unsigned unique,"+
			"i bigint,"+
			"c bigint,"+
			"m mediumtext,"+
			"index i (i))", tableName)
		if _, err := db.Exec(createTableStmt); err != nil {
			return err
		}
		for i := uint64(0); i < c.cfg.TableSize; i += batchSize {
			if _, err := db.Exec(insertSQL(tableName, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Execute implements Client Execute interface.
func (c *Client) Execute(ctx context.Context, db *sql.DB) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	wg := new(sync.WaitGroup)
	wg.Add(c.cfg.Concurrency)
	for i := 0; i < c.cfg.Concurrency; i++ {
		se, err := c.NewSession(db, uint64(i), c.cfg.TableSize)
		if err != nil {
			log.Fatal(err)
		}
		go se.Run(wg)
	}
	if !c.cfg.InsertDelete {
		go c.checkLoop(db)
	}
	go c.statsLoop()
	wg.Wait()
	return nil
}

func (c *Client) String() string {
	return caseName
}

func insertSQL(tableName string, beginID uint64) string {
	var values []string
	for i := beginID; i < beginID+batchSize; i++ {
		value := fmt.Sprintf("(%d, %d, %d, %d, '%s')", i, i, i, 0, "")
		values = append(values, value)
	}
	return fmt.Sprintf("insert %s values %s", tableName, strings.Join(values, ","))
}

// Session ...
type Session struct {
	seID           uint64
	isPessimistic  bool
	operationCount uint64
	conn           *sql.Conn
	stmts          []func(ctx context.Context) error
	ran            *randIDGenerator
	addedCount     int
	txnStart       time.Time
	commitStart    time.Time
	stmtCache      map[string]*sql.Stmt
	pCase          *Client
	scores         map[int]int
}

// NewSession ...
func (c *Client) NewSession(db *sql.DB, seID, maxSize uint64) (*Session, error) {
	ctx := context.Background()
	con, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	se := &Session{
		seID:           seID,
		conn:           con,
		operationCount: c.cfg.OperationCount / uint64(c.cfg.Concurrency),
		ran:            newRandIDGenerator(c.cfg.TableNum, maxSize),
		stmtCache:      make(map[string]*sql.Stmt),
		pCase:          c,
	}
	switch c.cfg.Mode {
	case "pessimistic":
		se.isPessimistic = true
	case "mix":
		se.isPessimistic = se.seID%2 == 1
	}
	se.stmts = append(se.stmts,
		se.insertIgnore,
		se.insertThenDelete,
		se.insertOnDuplicateUpdate,
		se.updateMultiTableSimple,
		se.updateMultiTableIndex,
		se.updateMultiTableRange,
		se.updateMultiTableIndexRange,
		se.updateMultiTableUniqueIndex,
		se.randSizeExceededTransaciton,
		se.replace,
		se.selectForUpdate,
		se.plainSelect,
	)
	return se, nil
}

func (se *Session) getAndCacheStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	if stmt, ok := se.stmtCache[query]; ok {
		return stmt, nil
	}

	stmt, err := se.conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	se.stmtCache[query] = stmt
	return stmt, nil
}

func (se *Session) runTransaction(parent context.Context) error {
	se.reset()
	ctx, cancel := context.WithTimeout(parent, time.Minute)
	defer cancel()
	beginSQL := "begin /*!90000 optimistic */"
	if se.isPessimistic {
		beginSQL = "begin /*!90000 pessimistic */"
	}
	_, err := se.conn.ExecContext(ctx, beginSQL)
	if err != nil {
		return err
	}
	numStmts := 1 + se.ran.uniform.Intn(5)
	for i := 0; i < numStmts; i++ {
		stmtType := se.ran.uniform.Intn(len(se.stmts))
		f := se.stmts[stmtType]
		err = f(ctx)
		if err != nil {
			se.handleError(ctx, err, false)
			return nil
		}
	}
	se.commitStart = time.Now()
	_, err = se.conn.ExecContext(ctx, "commit")
	if err != nil {
		se.handleError(ctx, err, true)
	} else {
		atomic.AddUint64(&se.pCase.successTxn, 1)
	}
	return nil
}

func (se *Session) runInsertDeleteTransaction(parent context.Context) error {
	se.reset()
	ctx, cancel := context.WithTimeout(parent, time.Minute)
	defer cancel()
	beginSQL := "begin /*!90000 optimistic */"
	if se.isPessimistic {
		beginSQL = "begin /*!90000 pessimistic */"
	}
	_, err := se.conn.ExecContext(ctx, beginSQL)
	if err != nil {
		return err
	}
	for i := 0; i < 4; i++ {
		rowID := se.ran.nextRowID()
		selectSQL := fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, 0)
		row := se.conn.QueryRowContext(ctx, selectSQL, rowID)
		var c int64
		err = row.Scan(&c)
		if err == sql.ErrNoRows {
			insertSQL := fmt.Sprintf("insert %s%d values (?, ?, ?, 0)", tableNamePrefix, 0)
			_, err = se.conn.ExecContext(ctx, insertSQL, rowID, rowID, rowID)
			if err != nil {
				if getErrorCode(err) != 1062 {
					log.Info("insert err", err)
				}
			}
		} else {
			deleteSQL := fmt.Sprintf("delete from %s%d where id = ?", tableNamePrefix, 0)
			_, err = se.conn.ExecContext(ctx, deleteSQL, rowID)
			if err != nil {
				log.Info("delete err", err)
			}
		}
	}
	se.commitStart = time.Now()
	_, err = se.conn.ExecContext(ctx, "commit")
	if err != nil {
		se.handleError(ctx, err, true)
	} else {
		atomic.AddUint64(&se.pCase.successTxn, 1)
	}
	return nil
}

func getErrorCode(err error) int {
	var code int
	_, err1 := fmt.Sscanf(err.Error(), "Error %d:", &code)
	if err1 != nil {
		return -1
	}
	return code
}

func (se *Session) handleError(ctx context.Context, err error, isCommit bool) {
	atomic.AddUint64(&se.pCase.failTxn, 1)
	_, _ = se.conn.ExecContext(ctx, "rollback")
	code := getErrorCode(err)
	ignoreCodes := se.pCase.cfg.IgnoreCodesO
	if se.isPessimistic {
		ignoreCodes = se.pCase.cfg.IgnoreCodesP
	}
	for _, ignoreCode := range ignoreCodes {
		if ignoreCode == code {
			return
		}
	}
	txnMode := "optimistic"
	if se.isPessimistic {
		txnMode = "pessimistic"
	}
	if isCommit {
		log.Info(txnMode, "txnDur", time.Since(se.txnStart), "commitDur", time.Since(se.commitStart), err)
	} else {
		log.Info(txnMode, se.isPessimistic, "txnDur", time.Since(se.txnStart), err)
	}
}

// Run ...
func (se *Session) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	ctx := context.Background()
	runFunc := se.runTransaction
	if se.pCase.cfg.InsertDelete {
		runFunc = se.runInsertDeleteTransaction
	}
	for i := uint64(0); i < se.operationCount; i++ {
		if err := runFunc(ctx); err != nil {
			log.Info("begin error", err)
			return
		}
	}
}

func (se *Session) executeDML(ctx context.Context, sqlFormat string, args ...interface{}) error {
	var res sql.Result
	var err error
	if se.pCase.cfg.UsePrepareStmt {
		stmt, err := se.getAndCacheStmt(ctx, sqlFormat)
		if err != nil {
			return err
		}
		res, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			return err
		}
	} else {
		res, err = se.conn.ExecContext(ctx, sqlFormat, args...)
		if err != nil {
			return err
		}
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		msg := fmt.Sprintf("affected row is 0, sqlFormat: %s, %v", sqlFormat, args)
		if len(msg) > 100 {
			msg = msg[:100] + "..."
		}
		return fmt.Errorf(msg)
	}
	se.addedCount += int(affected)
	return nil
}

func (se *Session) executeSelect(ctx context.Context, sqlFormat string, args ...interface{}) error {
	var rows *sql.Rows
	var err error
	if se.pCase.cfg.UsePrepareStmt {
		stmt, err := se.getAndCacheStmt(ctx, sqlFormat)
		if err != nil {
			return err
		}
		rows, err = stmt.QueryContext(ctx, args...)
		if err != nil {
			return err
		}
	} else {
		rows, err = se.conn.QueryContext(ctx, sqlFormat, args...)
		if err != nil {
			return err
		}
	}
	for rows.Next() {
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	return rows.Close()
}

func (se *Session) executeCommit(ctx context.Context) error {
	_, err := se.conn.ExecContext(ctx, "commit")
	return err
}

func (se *Session) reset() {
	se.ran.allocated = map[uint64]struct{}{}
	se.addedCount = 0
	se.txnStart = time.Now()
}

func (c *Client) statsLoop() {
	ticker := time.NewTicker(time.Second * 10)
	lastSuccess := uint64(0)
	lastFail := uint64(0)
	for {
		<-ticker.C
		curSuccess := atomic.LoadUint64(&c.successTxn)
		curFail := atomic.LoadUint64(&c.failTxn)
		log.Infof("tps(success:%v fail:%v)\n", float64(curSuccess-lastSuccess)/10, float64(curFail-lastFail)/10)
		lastSuccess = curSuccess
		lastFail = curFail
	}
}

func (c *Client) checkLoop(db *sql.DB) {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	ticker := time.NewTicker(time.Second * 30)
	tableNames := make([]string, c.cfg.TableNum)
	for i := 0; i < c.cfg.TableNum; i++ {
		tableNames[i] = fmt.Sprintf("select c from %s%d", tableNamePrefix, i)
	}
	allSumStmt := fmt.Sprintf("select sum(c) from (%s) tall", strings.Join(tableNames, " union all "))

	for {
		<-ticker.C
		checkCount(conn, allSumStmt, 0)
		for i := 0; i < c.cfg.TableNum; i++ {
			checkCount(conn, fmt.Sprintf("select count(*) from %s%d use index (i)", tableNamePrefix, i), int64(c.cfg.TableSize))
			checkCount(conn, fmt.Sprintf("select count(*) from %s%d use index (u)", tableNamePrefix, i), int64(c.cfg.TableSize))
		}
	}
}

func checkCount(conn *sql.Conn, sql string, expected int64) {
	row := conn.QueryRowContext(context.Background(), sql)
	var c int64
	if err := row.Scan(&c); err != nil {
		log.Info("check", sql, err)
	} else {
		if c != expected {
			panic(fmt.Sprintf("data inconsistency, %s is %d, expect %d", sql, c, expected))
		}
	}
}
