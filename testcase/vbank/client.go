package vbank

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
)

// primary key types
const (
	PKTypeInt     = "int"
	PKTypeDecimal = "decimal"
	PKTypeString  = "string"
)

// Config is the config of the test case.
type Config struct {
	// PKType
	PKType string
	// When Partition is set to true, we use multiple partition for multiple account instead of multiple table.
	Partition bool
	// When Range is set to true, we use range condition instead of equal condition which
	// leads to coprocessor request instead of point get request.
	Range bool
	// When ReadCommitted is set to true, we use read-committed isolation level for pessimistic transaction.
	ReadCommitted bool
	// When UpdateInPlace is set to true, we update with `set balance = balance - @amount`, otherwise we select for update first,
	// Then update by `set balance = @new_balance`.
	UpdateInPlace bool
	// ConnParams are parameters passed to the mysql-driver, see https://github.com/go-sql-driver/mysql#parameters .
	ConnParams string
}

const (
	vbAccountNum           = 10
	vbInitialBalance       = 20
	vbCreateInitialBalance = 10
)

func (c *Client) multiTable() bool { return !c.cfg.Partition }

func (c *Client) genCreateTableSQL(i int) string {
	var pkType string
	switch c.cfg.PKType {
	case PKTypeInt:
		pkType += "INT"
	case PKTypeDecimal:
		pkType += "DECIMAL(10, 2)"
	case PKTypeString:
		pkType += "VARCHAR(64)"
	default:
		panic("unknown PK type")
	}
	var partitionStr string
	if c.cfg.Partition {
		partitionStr = " PARTITION BY HASH(id) PARTITIONS 8"
	}
	createTableTemplate := `CREATE TABLE %s (
	id %s PRIMARY KEY,
	balance DECIMAL(10, 2),
	balance2 DECIMAL(10, 2) GENERATED ALWAYS AS (-balance) VIRTUAL,
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	INDEX i_balance (balance),
	INDEX i_balance2 (balance2),
	INDEX i_updated_at (updated_at)
)%s`
	return fmt.Sprintf(createTableTemplate, c.getTableName(i), pkType, partitionStr)
}

func (c *Client) genInitialInsertSQL(id int) string {
	rowID := id
	if c.multiTable() {
		rowID = 0
	}
	return fmt.Sprintf("insert into %s (id, balance, created_at) VALUES (%s, %d, '%.19s')",
		c.getTableName(id), c.idValue(rowID), vbInitialBalance, time.Now(),
	)
}

// Response ...
type Response struct {
	ReqType        VBReqType
	TS             uint64
	ReadResult     *BankState
	CreateResult   *CreateResult
	DeleteResult   *DeleteResult
	TransferResult *TransferResult
	OK             bool
	Err            string
}

// IsUnknown implements UnknownResponse interface
func (r *Response) IsUnknown() bool {
	return false
}

func (r *Response) setErr(err error) {
	r.Err = err.Error()
}

func (r *Response) String() string {
	var result, status string
	switch r.ReqType {
	case reqTypeRead:
		result = r.ReadResult.String()
	case reqTypeCreateAccount:
		result = r.CreateResult.String()
	case reqTypeDeleteAccount:
		result = r.DeleteResult.String()
	case reqTypeTransfer:
		result = r.TransferResult.String()
	}
	if r.OK {
		status = "ok"
	} else {
		status = r.Err
	}
	return fmt.Sprintf("%s %s ts:%d %s", r.ReqType, status, r.TS, result)
}

// Client implements the core.Client interface.
type Client struct {
	idx int
	r   *rand.Rand
	db  *sql.DB
	tx  *sql.Tx
	cfg *Config
}

// SetUp implements the core.Client interface.
func (c *Client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	c.idx = idx
	c.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
	if len(c.cfg.ConnParams) > 0 {
		dsn = dsn + "?" + c.cfg.ConnParams
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	c.db = db
	if idx != 0 {
		return nil
	}
	c.dropTables(ctx)
	tblCnt := vbAccountNum
	if !c.multiTable() {
		tblCnt = 1
	}
	_, err = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS v_bank_txn_status (ts BIGINT UNSIGNED PRIMARY KEY)")
	for i := 0; i < tblCnt; i++ {
		_, err = db.ExecContext(ctx, c.genCreateTableSQL(i))
		if err != nil {
			log.Error(c.logStr(err))
			return err
		}
	}
	for i := 0; i < vbAccountNum; i++ {
		_, err = db.ExecContext(ctx, c.genInitialInsertSQL(i))
		if err != nil {
			log.Error(c.logStr(err))
			return err
		}
	}
	return nil
}

// TearDown implements the core.Client interface.
func (c *Client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	if c.idx == 0 {
		c.dropTables(ctx)
	}
	return c.db.Close()
}

func (c *Client) dropTables(ctx context.Context) {
	if c.multiTable() {
		for i := 0; i < vbAccountNum; i++ {
			c.db.ExecContext(ctx, "drop table if exists "+c.getTableName(i))
		}
	} else {
		c.db.ExecContext(ctx, "drop table if exists v_bank")
	}
	c.db.ExecContext(ctx, "DROP TABLE IF EXISTS v_bank_txn_status")
}

// VBReqType ...
type VBReqType int

func (rt VBReqType) String() string {
	switch rt {
	case reqTypeRead:
		return "read"
	case reqTypeTransfer:
		return "transfer"
	case reqTypeCreateAccount:
		return "create_account"
	case reqTypeDeleteAccount:
		return "delete_account"
	}
	return "unknown"
}

const (
	reqTypeRead          VBReqType = 1
	reqTypeTransfer      VBReqType = 2
	reqTypeCreateAccount VBReqType = 3
	reqTypeDeleteAccount VBReqType = 4
)

// Invoke implements the core.Client interface.
func (c *Client) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	rt := r.(VBReqType)
	resp := &Response{}
	resp.ReqType = r.(VBReqType)
	var err error
	resp.TS, err = c.beginTxn(ctx, rt)
	if err != nil {
		resp.setErr(err)
		return resp
	}
	switch r.(VBReqType) {
	case reqTypeRead:
		resp.ReadResult, err = c.invokeRead(ctx)
	case reqTypeTransfer:
		resp.TransferResult, err = c.invokeTransfer(ctx)
	case reqTypeCreateAccount:
		resp.CreateResult, err = c.invokeCreateAccount(ctx)
	case reqTypeDeleteAccount:
		resp.DeleteResult, err = c.invokeDeleteAccount(ctx)
	}
	c.endTxn(ctx, resp, err)
	return resp
}

func (c *Client) beginTxn(ctx context.Context, reqType VBReqType) (uint64, error) {
	opts := &sql.TxOptions{}
	if c.cfg.ReadCommitted && reqType != reqTypeRead {
		opts.Isolation = sql.LevelReadCommitted
	}
	tx, err := c.db.BeginTx(ctx, opts)
	if err != nil {
		log.Error(c.logStr(err))
		return 0, err
	}
	var ts uint64
	err = tx.QueryRowContext(ctx, "SELECT @@tidb_current_ts").Scan(&ts)
	if err != nil {
		log.Error(c.logStr(err))
		tx.Rollback()
		return 0, err
	}
	c.tx = tx
	return ts, err
}

func (c *Client) endTxn(ctx context.Context, resp *Response, err error) {
	if err != nil {
		resp.setErr(err)
		err = c.tx.Rollback()
		if err != nil {
			log.Error(c.logStr(err))
		}
	} else {
		if resp.ReqType != reqTypeRead {
			_, err = c.tx.ExecContext(ctx, fmt.Sprintf("INSERT v_bank_txn_status VALUES (%d)", resp.TS))
			if err != nil {
				resp.setErr(err)
				c.tx.Rollback()
				return
			}
		}
		err = c.tx.Commit()
		if err != nil {
			log.Error(c.logStr(err))
			resp.OK = c.checkTxnStatus(ctx, resp.TS)
			if !resp.OK {
				resp.setErr(err)
			}
		} else {
			resp.OK = true
		}
	}
}

func (c *Client) checkTxnStatus(ctx context.Context, ts uint64) (committed bool) {
	for {
		var x uint64
		err := c.db.QueryRowContext(ctx, fmt.Sprintf("SELECT ts FROM v_bank_txn_status WHERE ts = %d", ts)).Scan(&x)
		if err == nil {
			return true
		}
		if err == sql.ErrNoRows {
			return false
		}
		log.Error(c.logStr(err))
		select {
		case <-ctx.Done():
			return false
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (c *Client) getTableName(accID int) string {
	if c.multiTable() {
		return "v_bank_" + strconv.Itoa(accID)
	}
	return "v_bank"
}

func (c *Client) getWhereClause(accID int) string {
	if !c.multiTable() {
		if c.cfg.Range {
			return fmt.Sprintf("id > %s and id < %s", c.idValue(accID-1), c.idValue(accID+1))
		}
		return fmt.Sprintf("id = %s", c.idValue(accID))
	}
	if c.cfg.Range {
		return fmt.Sprintf("id >= %s", c.idValue(0))
	}
	return fmt.Sprintf("id = %s", c.idValue(0))
}

func (c *Client) idValue(id int) string {
	if c.multiTable() {
		id = 0
	}
	if c.cfg.PKType == PKTypeString {
		return fmt.Sprintf("'%d'", id)
	}
	return strconv.Itoa(id)
}

// BankState is the state of the test case.
type BankState struct {
	Accounts []AccountState
}

func (bs *BankState) equal(bs2 *BankState) bool {
	if len(bs.Accounts) != len(bs2.Accounts) {
		return false
	}
	for i := range bs.Accounts {
		if bs.Accounts[i] != bs2.Accounts[i] {
			return false
		}
	}
	return true
}

func (bs *BankState) transfer(from, to int, amount float64) {
	if from == to {
		return
	}
	for i := range bs.Accounts {
		acc := &bs.Accounts[i]
		if acc.ID == from {
			acc.Balance -= amount
		} else if acc.ID == to {
			acc.Balance += amount
		}
	}
}

func (bs *BankState) createAccount(newID int) {
	bs.Accounts[0].Balance -= vbCreateInitialBalance
	bs.append(newID, vbCreateInitialBalance)
	sort.Slice(bs.Accounts, func(i, j int) bool {
		return bs.Accounts[i].ID < bs.Accounts[j].ID
	})
}

func (bs *BankState) deleteAccount(victimID int, balance float64) {
	bs.Accounts[0].Balance += balance
	for i := range bs.Accounts {
		if bs.Accounts[i].ID == victimID {
			bs.Accounts = append(bs.Accounts[:i], bs.Accounts[i+1:]...)
			break
		}
	}
}

// AccountState ...
type AccountState struct {
	ID      int
	Balance float64
}

func (bs *BankState) sum() float64 {
	var x float64
	for _, v := range bs.Accounts {
		x += v.Balance
	}
	return x
}

func (bs *BankState) String() string {
	if bs == nil {
		return ""
	}
	var parts []string
	var sum float64
	for _, accState := range bs.Accounts {
		parts = append(parts, fmt.Sprintf("%d:%.0f", accState.ID, accState.Balance))
		sum += accState.Balance
	}
	return fmt.Sprintf("read_result(sum:%.0f,%s)", sum, strings.Join(parts, ","))
}

// Clone clones the state.
func (bs *BankState) Clone() *BankState {
	n := &BankState{}
	n.Accounts = append(n.Accounts, bs.Accounts...)
	return n
}

func (bs *BankState) append(id int, balance float64) {
	bs.Accounts = append(bs.Accounts, AccountState{
		ID:      id,
		Balance: balance,
	})
}

func (c *Client) invokeRead(ctx context.Context) (*BankState, error) {
	if c.multiTable() {
		return c.invokeReadMultiTable(ctx)
	}
	return c.invokeReadSingleTable(ctx)
}

func (c *Client) invokeReadMultiTable(ctx context.Context) (result *BankState, err error) {
	result = &BankState{}
	for i := 0; i < vbAccountNum; i++ {
		var balance float64
		err = c.tx.QueryRowContext(ctx, "select balance from "+c.getTableName(i)).Scan(&balance)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			log.Error(c.logStr(err))
			return
		}
		result.append(i, balance)
	}
	return
}

func (c *Client) invokeReadSingleTable(ctx context.Context) (result *BankState, err error) {
	result = &BankState{}
	rows, err := c.tx.QueryContext(ctx, "select CONVERT(id, SIGNED INTEGER), balance from v_bank order by id")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var balance float64
		err = rows.Scan(&id, &balance)
		if err != nil {
			log.Error(err)
			return
		}
		result.append(id, balance)
	}
	return
}

// TransferResult ...
type TransferResult struct {
	Aborted     bool
	FromID      int
	FromBalance float64
	ToID        int
	ToBalance   float64
	Amount      float64
}

func (tr *TransferResult) String() string {
	if tr == nil {
		return ""
	}
	if tr.Aborted {
		return "transfer_aborted"
	}
	return fmt.Sprintf("transfer_ok(from:%d:%.0f, to:%d:%.0f, amount:%.0f)",
		tr.FromID, tr.FromBalance, tr.ToID, tr.ToBalance, tr.Amount)
}

const (
	selectForUpdateFmt = "SELECT balance FROM %s WHERE %s FOR UPDATE"
	updateFmt          = "UPDATE %s SET balance = %.0f WHERE %s"
	updateInPlaceFmt   = "UPDATE %s SET balance = balance + %.0f WHERE %s"
)

func (c *Client) invokeTransfer(ctx context.Context) (*TransferResult, error) {
	if c.cfg.UpdateInPlace {
		return c.invokeTransferUpdateInPlace(ctx)
	}
	return c.invokeTransferSelectForUpdate(ctx)
}

func (c *Client) invokeTransferSelectForUpdate(ctx context.Context) (r *TransferResult, err error) {
	r = &TransferResult{}
	r.FromID = c.r.Intn(vbAccountNum)
	r.ToID = c.r.Intn(vbAccountNum)
	if r.FromID == r.ToID {
		r.ToID = (r.ToID + 1) % vbAccountNum
	}
	r.Amount = float64(c.r.Intn(10) + 1)
	selectFromForUpdate := fmt.Sprintf(
		"SELECT balance FROM %s WHERE %s FOR UPDATE",
		c.getTableName(r.FromID), c.getWhereClause(r.FromID))
	err = c.tx.QueryRowContext(ctx, selectFromForUpdate).Scan(&r.FromBalance)
	if err == sql.ErrNoRows {
		// Since we never modify anything, we can commit the transaction.
		r.Aborted = true
		err = nil
		return
	}
	if err != nil {
		return
	}
	if r.FromBalance < r.Amount {
		r.Aborted = true
		err = nil
		return
	}
	r.FromBalance -= r.Amount
	selectToForUpdate := fmt.Sprintf(
		"SELECT balance FROM %s WHERE %s FOR UPDATE",
		c.getTableName(r.ToID), c.getWhereClause(r.ToID))
	err = c.tx.QueryRowContext(ctx, selectToForUpdate).Scan(&r.ToBalance)
	if err == sql.ErrNoRows {
		r.Aborted = true
		err = nil
		return
	}
	if err != nil {
		return
	}
	r.ToBalance += r.Amount
	updateFrom := fmt.Sprintf(updateFmt,
		c.getTableName(r.FromID), r.FromBalance, c.getWhereClause(r.FromID))
	execRs, err := c.tx.ExecContext(ctx, updateFrom)
	if err != nil {
		return
	}
	if affected, _ := execRs.RowsAffected(); affected != 1 {
		err = errors.New("unexpected affected rows " + strconv.Itoa(int(affected)))
		return
	}
	updateTo := fmt.Sprintf(updateFmt,
		c.getTableName(r.ToID), r.ToBalance, c.getWhereClause(r.ToID))
	execRs, err = c.tx.ExecContext(ctx, updateTo)
	if err != nil {
		return
	}
	if affected, _ := execRs.RowsAffected(); affected != 1 {
		err = errors.New("unexpected affected rows " + strconv.Itoa(int(affected)))
		return
	}
	return
}

func (c *Client) invokeTransferUpdateInPlace(ctx context.Context) (r *TransferResult, err error) {
	r = &TransferResult{}
	r.FromID = c.r.Intn(vbAccountNum)
	r.ToID = c.r.Intn(vbAccountNum)
	r.Amount = float64(c.r.Intn(10) + 1)
	updateFrom := fmt.Sprintf(
		updateInPlaceFmt+" AND balance >= %.0f",
		c.getTableName(r.FromID), -r.Amount, c.getWhereClause(r.FromID), r.Amount)
	execRs, err := c.tx.ExecContext(ctx, updateFrom)
	if err != nil {
		return
	}
	affected, err := execRs.RowsAffected()
	if err != nil {
		return
	}
	if affected == 0 {
		r.Aborted = true
		return r, nil
	}
	updateTo := fmt.Sprintf(
		updateInPlaceFmt,
		c.getTableName(r.ToID), r.Amount, c.getWhereClause(r.ToID))
	execRs, err = c.tx.ExecContext(ctx, updateTo)
	if err != nil {
		return
	}
	affected, err = execRs.RowsAffected()
	if affected == 0 {
		// The to account is not found, we need to rollback
		err = errors.New("to account not found")
		return
	}
	selectToForUpdate := fmt.Sprintf(selectForUpdateFmt, c.getTableName(r.ToID), c.getWhereClause(r.ToID))
	err = c.tx.QueryRowContext(ctx, selectToForUpdate).Scan(&r.ToBalance)
	if err != nil {
		return
	}
	selectFromForUpdate := fmt.Sprintf(selectForUpdateFmt, c.getTableName(r.FromID), c.getWhereClause(r.FromID))
	err = c.tx.QueryRowContext(ctx, selectFromForUpdate).Scan(&r.FromBalance)
	return
}

// DeleteResult ...
type DeleteResult struct {
	VictimID int
	Balance  float64
	Aborted  bool
}

func (dr *DeleteResult) String() string {
	if dr == nil {
		return ""
	}
	if dr.Aborted {
		return "delete_aborted"
	}
	return fmt.Sprintf("delete_ok(%d:%.0f)", dr.VictimID, dr.Balance)
}

func (c *Client) invokeDeleteAccount(ctx context.Context) (res *DeleteResult, err error) {
	res = &DeleteResult{}
	// we never delete account 0
	res.VictimID = c.r.Intn(9) + 1
	selectVictimForUpdate := fmt.Sprintf(selectForUpdateFmt, c.getTableName(res.VictimID), c.getWhereClause(res.VictimID))
	err = c.tx.QueryRowContext(ctx, selectVictimForUpdate).Scan(&res.Balance)
	if err == sql.ErrNoRows {
		res.Aborted = true
		return res, nil
	}
	deleteStmt := fmt.Sprintf("DELETE FROM %s WHERE %s", c.getTableName(res.VictimID), c.getWhereClause(res.VictimID))
	_, err = c.tx.ExecContext(ctx, deleteStmt)
	if err != nil {
		return
	}
	updateStmt := fmt.Sprintf(updateInPlaceFmt,
		c.getTableName(0), res.Balance, c.getWhereClause(0))
	_, err = c.tx.ExecContext(ctx, updateStmt)
	return
}

// CreateResult ...
type CreateResult struct {
	NewID   int
	Balance float64
}

func (cr *CreateResult) String() string {
	if cr == nil {
		return ""
	}
	return fmt.Sprintf("create(%d)", cr.NewID)
}

func (c *Client) invokeCreateAccount(ctx context.Context) (resp *CreateResult, err error) {
	resp = &CreateResult{}
	resp.NewID = c.r.Intn(9) + 1
	resp.Balance = vbCreateInitialBalance
	insertStmt := fmt.Sprintf("INSERT INTO %s (id, balance) VALUES (%s, %d)",
		c.getTableName(resp.NewID), c.idValue(resp.NewID), vbCreateInitialBalance)
	_, err = c.tx.ExecContext(ctx, insertStmt)
	if err != nil {
		return
	}
	selectZeroForUpdate := fmt.Sprintf(selectForUpdateFmt, c.getTableName(0), c.getWhereClause(0))
	var balance float64
	err = c.tx.QueryRowContext(ctx, selectZeroForUpdate).Scan(&balance)
	if err != nil {
		return
	}
	if balance < 5 {
		err = errors.New("account 0 balance too low to create new account")
		return
	}
	// take the balance from account 0.
	updateStmt := fmt.Sprintf(updateInPlaceFmt, c.getTableName(0), -resp.Balance, c.getWhereClause(0))
	_, err = c.tx.ExecContext(ctx, updateStmt)
	return
}

// NextRequest implements the core.Client interface.
func (c *Client) NextRequest() interface{} {
	//return reqTypeRead
	x := c.r.Intn(100)
	if x < 3 {
		return reqTypeDeleteAccount
	}
	if x < 12 {
		// 3 times possibility so we can have enough accounts.
		return reqTypeCreateAccount
	}
	if x < 50 {
		return reqTypeRead
	}
	return reqTypeTransfer
}

// DumpState implements the core.Client interface.
func (c *Client) DumpState(ctx context.Context) (interface{}, error) {
	resp := &Response{}
	var err error
	resp.TS, err = c.beginTxn(ctx, reqTypeRead)
	if err != nil {
		return nil, err
	}
	resp.ReadResult, err = c.invokeRead(ctx)
	c.endTxn(ctx, resp, err)
	return resp.ReadResult, err
}

func (c *Client) logStr(err error) string {
	return fmt.Sprintf("client(%d) error %s", c.idx, err.Error())
}

// ClientCreator creates a test client.
type ClientCreator struct {
	cfg *Config
}

// NewClientCreator creates a new ClientCreator.
func NewClientCreator(cfg *Config) *ClientCreator {
	return &ClientCreator{cfg: cfg}
}

// Create creates a Client.
func (cc *ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &Client{cfg: cc.cfg}
}

var _ core.Model = &Model{}

// Model implements the core.Model interface.
type Model struct {
	state *BankState
}

// Prepare implements the core.Model interface.
func (m *Model) Prepare(state interface{}) {
	m.state = state.(*BankState)
}

// Init implements the core.Model interface.
func (m *Model) Init() interface{} {
	if m.state != nil {
		return m.state
	}
	state := &BankState{}
	for i := 0; i < vbAccountNum; i++ {
		state.append(i, vbInitialBalance)
	}
	return state
}

// Step implements the core.Model interface.
func (m *Model) Step(state interface{}, input interface{}, output interface{}) (bool, interface{}) {
	st := state.(*BankState)
	reqType := input.(VBReqType)
	resp := output.(*Response)
	if !resp.OK {
		return true, st
	}
	if reqType == reqTypeRead {
		return st.equal(resp.ReadResult), st
	}
	newState := st.Clone()
	switch reqType {
	case reqTypeTransfer:
		res := resp.TransferResult
		if res.Aborted {
			return true, st
		}
		newState.transfer(res.FromID, res.ToID, res.Amount)
	case reqTypeCreateAccount:
		res := resp.CreateResult
		newState.createAccount(res.NewID)
	case reqTypeDeleteAccount:
		res := resp.DeleteResult
		if res.Aborted {
			return true, st
		}
		newState.deleteAccount(res.VictimID, res.Balance)
	}
	return true, newState
}

// Equal implements the core.Model interface.
func (m *Model) Equal(state1, state2 interface{}) bool {
	rs1 := state1.(*BankState)
	rs2 := state2.(*BankState)
	return rs1.equal(rs2)
}

// Name implements the core.Model interface.
func (m *Model) Name() string {
	return "v_bank"
}

var _ history.RecordParser = &Parser{}

// Parser implements the core.Parser interface.
type Parser struct{}

// OnRequest implements the core.Parser interface.
func (p Parser) OnRequest(data json.RawMessage) (interface{}, error) {
	var req VBReqType
	err := json.Unmarshal(data, &req)
	return req, err
}

// OnResponse implements the core.Parser interface.
func (p Parser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := &Response{}
	err := json.Unmarshal(data, r)
	return r, err
}

// OnNoopResponse implements the core.Parser interface.
func (p Parser) OnNoopResponse() interface{} {
	return &Response{}
}

// OnState implements the core.Parser interface.
func (p Parser) OnState(state json.RawMessage) (interface{}, error) {
	readResult := &BankState{}
	err := json.Unmarshal(state, readResult)
	return readResult, err
}
