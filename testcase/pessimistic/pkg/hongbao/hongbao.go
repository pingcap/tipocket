package hongbao

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
)

// CaseName ...
const CaseName = "HongBao"

// ClientConfig ...
type ClientConfig struct {
	DBName         string
	Concurrency    int
	UserNum        int
	FriendNum      int
	GroupNum       int
	GroupMemberNum int
	HongbaoNum     int
	IgnoreCodesO   []int
	IgnoreCodesP   []int
	TxnMode        string
}

// Client is for hongbao transaction test
type Client struct {
	cfg *ClientConfig

	successNum int32
	failNum    int32
}

// NewHongbaoCase ...
func NewHongbaoCase(cfg *ClientConfig) *Client {
	return &Client{cfg: cfg}
}

// Initialize ...
func (h *Client) Initialize(ctx context.Context, db *sql.DB) error {
	if err := InitSchema(ctx, db); err != nil {
		return err
	}
	return h.DoUsersTask(ctx, db, h.cfg.UserNum, h.cfg.FriendNum, h.cfg.GroupNum, h.cfg.GroupMemberNum, 0)
}

// Execute implements Client Execute interface.
func (h *Client) Execute(ctx context.Context, db *sql.DB) error {
	if err := h.DoHongbaoTasks(ctx, db, h.cfg.HongbaoNum, 0); err != nil {
		return err
	}
	return h.DoValidate(ctx, db)
}

// DoUsersTask ...
func (h *Client) DoUsersTask(ctx context.Context, db *sql.DB, userNum, friendNum, groupNum, groupMemberNum, sleep int) error {
	log.Info("Start user tasks")
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	se, err := NewSession(h, conn, 0, 0, 0, 0)
	if err != nil {
		return err
	}
	return se.CreateUsers(ctx, userNum, friendNum, groupNum, groupMemberNum, sleep)
}

// DoHongbaoTasks ...
func (h *Client) DoHongbaoTasks(ctx context.Context, db *sql.DB, hongbaoNum, sleep int) error {
	log.Infof("[%s] started", h)
	wg := new(sync.WaitGroup)
	wg.Add(h.cfg.Concurrency)

	sqlSelectUser := "select min(uid) as begin_uid, max(uid) as end_uid from `user`"
	var beginUserID int64
	var endUserID int64
	row := db.QueryRowContext(ctx, sqlSelectUser)
	if err := row.Scan(&beginUserID, &endUserID); err != nil {
		return err
	}

	conns := make([]*sql.Conn, h.cfg.Concurrency)
	var err error
	for i := 0; i < h.cfg.Concurrency; i++ {
		if conns[i], err = db.Conn(ctx); err != nil {
			log.Fatal(err)
		}
	}
	defer func() {
		for i := 0; i < h.cfg.Concurrency; i++ {
			if err = conns[i].Close(); err != nil {
				log.Fatal(err)
			}
		}
	}()
	for i := 0; i < h.cfg.Concurrency; i++ {
		se, err := NewSession(h, conns[i], beginUserID, endUserID, hongbaoNum, 0)
		if err != nil {
			return err
		}
		go se.DoHongbaoTask(ctx, wg)
	}
	wg.Wait()
	log.Infof("[%s] test end, succeeded txn: %d, failed txn: %d", h, h.successNum, h.failNum)
	return nil
}

// DoValidate ...
func (h *Client) DoValidate(ctx context.Context, db *sql.DB) error {
	sql := `
	select
		(select 100000 * (select count(*) from user)) as init_total_balance,
		(select sum(balance) from user) + (select sum(balance) from user_bank) as final_total_balance,
		(select sum(amount) from hongbao) as hb_amount,
		(select sum(amount) from hongbao_detail) as hb_dt_amount,
		(select sum(friends) friends from user) as user_friends,
		(select count(*) friends from user_friends) as uf_count,
		(select sum(group_members) group_members from group_info) as group_members,
		(select count(*) friends from group_member) as gm_count`
	var totalBalance int64
	var finalTotalBalance int64
	var hbAmount int64
	var hbDetailAmount int64
	var userFriends int64
	var ufCount int64
	var groupMembers int64
	var gmCount int64
	row := db.QueryRowContext(ctx, sql)
	err := row.Scan(&totalBalance, &finalTotalBalance, &hbAmount, &hbDetailAmount, &userFriends, &ufCount, &groupMembers, &gmCount)
	if err != nil {
		return err
	}
	if totalBalance != finalTotalBalance {
		return errors.New(fmt.Sprintf("balance not equal, total balance: %d, final balance: %d", totalBalance, finalTotalBalance))
	}
	if hbDetailAmount != hbAmount {
		return errors.New(fmt.Sprintf("hongbao amount not equal, amount: %d, detailed amount: %d", hbAmount, hbDetailAmount))
	}
	log.Infof("Validate done:")
	log.Infof("Total Balance: %d", totalBalance)
	log.Infof("Final Total Balance: %d", totalBalance)
	log.Infof("Hongbao Amount: %d", hbAmount)
	log.Infof("Hongbao Detail Amount: %d", hbDetailAmount)
	log.Infof("User Friends: %d", userFriends)
	log.Infof("UF Count: %d", ufCount)
	log.Infof("Group Members: %d", groupMembers)
	log.Infof("GM Count: %d", gmCount)
	return nil
}

func (h *Client) String() string {
	return CaseName
}
