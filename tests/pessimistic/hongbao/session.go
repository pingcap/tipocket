package hongbao

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"

	_ "github.com/go-sql-driver/mysql" // mysql
)

const pessimisticBegin = "begin /*!90000 pessimistic */"
const optimisticBegin = "begin /*!90000 optimistic */"

// Session ...
type Session struct {
	conn          *sql.Conn
	hongbaoClient *Client
	beginUserID   int64
	endUserID     int64
	hongbaoNum    int
	sleep         int

	successNum int32
	failNum    int32
}

// NewSession ...
func NewSession(
	hongbaoClient *Client,
	conn *sql.Conn,
	beginUserID int64,
	endUserID int64,
	hongbaoNum,
	sleep int) (*Session, error) {
	if endUserID < beginUserID {
		return nil, errors.New(fmt.Sprintf("endUserID[%d] must be greater than beginUserID[%d]", endUserID, beginUserID))
	}
	se := &Session{
		hongbaoClient: hongbaoClient,
		conn:          conn,
		beginUserID:   beginUserID,
		endUserID:     endUserID,
		hongbaoNum:    hongbaoNum,
		sleep:         sleep,
	}
	return se, nil
}

// BeginTxn ...
func (s *Session) BeginTxn(ctx context.Context) error {
	var beginStmt string
	switch s.hongbaoClient.cfg.TxnMode {
	case "pessimistic":
		beginStmt = pessimisticBegin
	case "optimistic":
		beginStmt = optimisticBegin
	case "mix":
		if rand.Intn(2) == 0 {
			beginStmt = pessimisticBegin
		} else {
			beginStmt = optimisticBegin
		}
	default:
		panic(fmt.Sprintf("Invalid transaction mode: %s", s.hongbaoClient.cfg.TxnMode))
	}
	_, err := s.conn.ExecContext(ctx, beginStmt)
	return err
}

// CommitTxn ...
func (s *Session) CommitTxn(ctx context.Context) error {
	if _, err := s.conn.ExecContext(ctx, "commit"); err != nil {
		if err2 := s.RollbackTxn(ctx); err2 != nil {
			return err2
		}
		return err
	}
	return nil
}

// RollbackTxn ...
func (s *Session) RollbackTxn(ctx context.Context) error {
	if _, err := s.conn.ExecContext(ctx, "rollback"); err != nil {
		return err
	}
	return nil
}

// CreateUser ...
func (s *Session) CreateUser(ctx context.Context, balance int) (userID int64, err error) {
	sqlInsertUser := "insert into `user` " +
		"set uname = ?, birth_day = ?, addr_province = ?, addr_city = ?, friends = 0 "
	sqlInsertUserBank := "insert into `user_bank` set uid = ?, balance = ?"

	r, err := s.conn.ExecContext(ctx, sqlInsertUser, randUserName(), randBirthday(), randProvinceID(), randCityID())
	if err != nil {
		return -1, err
	}
	rowID, err := r.LastInsertId()
	if err != nil {
		return -1, err
	}

	_, err = s.conn.ExecContext(ctx, sqlInsertUserBank, rowID, balance)
	if err != nil {
		return -1, err
	}
	return rowID, err
}

// CreateFriends ...
func (s *Session) CreateFriends(ctx context.Context, userID int64, count int) error {
	sqlQueryExistingFriends := `select count(ufid) from user_friends a where uid = ?`
	sqlQueryPickFriends := `
				select uid from user a where uid != ? and uid not in (
	           select ufid from user_friends b where b.uid = ?) order by rand() limit ?
	   `
	sqlInsertUserFriends := "insert into user_friends set uid = ?, ufid = ?"
	sqlUpdateUser := "update `user` set  friends = friends + ? where uid = ?"

	var existingFriends int
	row := s.conn.QueryRowContext(ctx, sqlQueryExistingFriends, userID)
	if err := row.Scan(&existingFriends); err != nil {
		return err
	}
	remainedCount := count - existingFriends
	if remainedCount <= 0 {
		// 朋友数已经够了
		return nil
	}
	friendSlice := make([]int64, 0, remainedCount)
	{
		rows, err := s.conn.QueryContext(ctx, sqlQueryPickFriends, userID, userID, remainedCount)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var friendID int64
			if err = rows.Scan(&friendID); err != nil {
				return err
			}
			friendSlice = append(friendSlice, friendID)
		}
	}
	for _, friendID := range friendSlice {
		if _, err := s.conn.ExecContext(ctx, sqlInsertUserFriends, userID, friendID); err != nil {
			return err
		}
		if _, err := s.conn.ExecContext(ctx, sqlInsertUserFriends, friendID, userID); err != nil {
			return err
		}
		if _, err := s.conn.ExecContext(ctx, sqlUpdateUser, 1, friendID); err != nil {
			return err
		}
	}
	if _, err := s.conn.ExecContext(ctx, sqlUpdateUser, len(friendSlice), userID); err != nil {
		return err
	}
	return nil
}

// CreateGroup ...
func (s *Session) CreateGroup(ctx context.Context, userID int64, members int) (groupID int64, err error) {
	// 获取用户好友,及好友的好友,并随机选择一部分后,去掉重复的id
	sqlQueryUserFriendsByFriends := ` 
		select distinct(muid) from 
		 (select ufid as muid from user_friends where uid = ?
		   union
		  select b.ufid as muid from user_friends a, user_friends b where a.ufid = b.uid and a.uid = ?
		 ) t
		where muid != ? order by rand() limit ?;
        `
	sqlInsertGroup := "insert into `group_info` set create_uid = ?, gname = ?"
	sqlUpdateGroup := "update `group_info` set group_members = group_members + ? where gid = ?"
	sqlInsertGroupMember := "insert into `group_member` set gid = ?, uid = ?"

	userFriendsByFriends := make([]int64, 0, members)
	{
		rows, err := s.conn.QueryContext(ctx, sqlQueryUserFriendsByFriends, userID, userID, userID, members)
		if err != nil {
			return -1, err
		}
		defer rows.Close()
		for rows.Next() {
			var memberID int64
			if err = rows.Scan(&memberID); err != nil {
				return -1, err
			}
			userFriendsByFriends = append(userFriendsByFriends, memberID)
		}
	}
	res, err := s.conn.ExecContext(ctx, sqlInsertGroup, userID, randGroupName())
	if err != nil {
		return -1, err
	}
	groupID, err = res.LastInsertId()
	if err != nil {
		return -1, err
	}
	for _, memberID := range userFriendsByFriends {
		if _, err := s.conn.ExecContext(ctx, sqlInsertGroupMember, groupID, memberID); err != nil {
			return -1, err
		}
	}
	_, err = s.conn.ExecContext(ctx, sqlUpdateGroup, len(userFriendsByFriends), groupID)
	return groupID, err
}

// UserAddBalance ...
func (s *Session) UserAddBalance(ctx context.Context, userID int64, amount int64) error {
	sqlUpdateUserBank := "update `user_bank` set balance=balance - ? where uid = ?"
	sqlUpdateUser := "update `user` set balance=balance + ? where uid = ?"

	_, err := s.conn.ExecContext(ctx, sqlUpdateUserBank, amount, userID)
	if err != nil {
		return err
	}
	_, err = s.conn.ExecContext(ctx, sqlUpdateUser, amount, userID)
	return err
}

// UserBankAddBalance ...
func (s *Session) UserBankAddBalance(ctx context.Context, txn *sql.Tx, userID int64, amount int64) error {
	sqlUpdateUserBank := "update `user_bank` set balance=balance + ? where uid = ?"
	sqlUpdateUser := "update `user` set  balance=balance - ? where uid = ?"
	_, err := txn.ExecContext(ctx, sqlUpdateUserBank, amount, userID)
	if err != nil {
		return err
	}
	_, err = txn.ExecContext(ctx, sqlUpdateUser, amount, userID)
	return err
}

// CreateUsers ...
func (s *Session) CreateUsers(ctx context.Context, userNum, friendNum, groupNum, groupMemberNum, sleep int) error {
	userIDs := make([]int64, userNum)
	log.Infof("[%s] Creating %d users", CaseName, userNum)
	for i := 0; i < userNum; i++ {
		if err := s.BeginTxn(ctx); err != nil {
			return err
		}
		userID, err := s.CreateUser(ctx, 100000)
		if err != nil {
			if err2 := s.RollbackTxn(ctx); err2 != nil {
				log.Fatal(err2)
			}
			return err
		}
		if err := s.CommitTxn(ctx); err != nil {
			return err
		}
		userIDs[i] = userID
	}

	log.Infof("[%s] Creating %d friends for each user", CaseName, friendNum)
	for _, id := range userIDs {
		if err := s.BeginTxn(ctx); err != nil {
			return err
		}
		if err := s.CreateFriends(ctx, id, friendNum); err != nil {
			log.Infof("Error creating friends for user: %v, rollback", id)
			if err2 := s.RollbackTxn(ctx); err2 != nil {
				log.Fatal(err2)
			}
			return err
		}
		if err := s.CommitTxn(ctx); err != nil {
			log.Infof("Error commiting friends for user: %v, rollback", id)
			return err
		}
	}

	log.Infof("[%s] Creating %d groups for each user", CaseName, groupNum)
	for _, id := range userIDs {
		for j := 0; j < groupNum; j++ {
			if err := s.BeginTxn(ctx); err != nil {
				return err
			}
			_, err := s.CreateGroup(ctx, id, groupMemberNum)
			if err != nil {
				if err2 := s.RollbackTxn(ctx); err2 != nil {
					log.Fatal(err2)
				}
				return err
			}
			if err := s.CommitTxn(ctx); err != nil {
				return err
			}
			if sleep > 0 {
				time.Sleep(time.Duration(sleep) * time.Second)
			}
		}
	}
	return nil
}

// DoHongbaoTask ...
func (s *Session) DoHongbaoTask(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := 0; j < s.hongbaoNum; j++ {
		userID := s.beginUserID + rand.Int63n(s.endUserID-s.beginUserID)
		if err := s.CreateHongbao(ctx, userID, 100); err != nil {
			log.Errorf("Error while creating hongbao: %v", err)
			s.failNum++
		} else {
			s.successNum++
		}
		if s.sleep > 0 {
			time.Sleep(time.Duration(s.sleep) * time.Millisecond)
		}
	}
	atomic.AddInt32(&s.hongbaoClient.successNum, s.successNum)
	atomic.AddInt32(&s.hongbaoClient.failNum, s.failNum)
}

// CreateHongbao ...
func (s *Session) CreateHongbao(ctx context.Context, userID int64, amount int64) error {
	// 发红包
	sqlQueryUser := "select balance from `user` where uid = ?"
	sqlQueryGroup := "select gid from `group_member` where uid = ?"
	sqlQueryGroupMember := "select uid from `group_member` where gid = ? order by rand()"
	// sqlQueryHongbao := "select * from `hongbao` where reid = ?"
	sqlInsertHongbao := "insert into hongbao set uid = ?, gid = ?, amount = ?"
	sqlInsertHongbaoDetail := "insert into hongbao_detail set reid = ?, uid = ?, amount = ?"
	sqlUpdateUser := "update `user` set balance = balance + ? where uid = ?"
	sqlUpdateHongbao := "update hongbao set best_luck_uid = ?, max_mount = ?, pickup_users = ? where reid = ?"

	// 检查帐户余额,没有就先冲值
	row := s.conn.QueryRowContext(ctx, sqlQueryUser, userID)
	var balance int64
	if err := row.Scan(&balance); err != nil {
		return err
	}
	if balance < amount {
		if err := s.BeginTxn(ctx); err != nil {
			return err
		}
		if err := s.UserAddBalance(ctx, userID, amount); err != nil {
			if err2 := s.RollbackTxn(ctx); err2 != nil {
				log.Fatal(err2)
			}
			return err
		}
		if err := s.CommitTxn(ctx); err != nil {
			return err
		}
	}
	// 随机选择一个群gid
	rows, err := s.conn.QueryContext(ctx, sqlQueryGroup, userID)
	if err != nil {
		return err
	}
	groupList := make([]int64, 0, 10)
	var groupID int64
	for rows.Next() {
		if err = rows.Scan(&groupID); err != nil {
			return err
		}
		groupList = append(groupList, groupID)
	}
	groupID = groupList[rand.Intn(len(groupList))]

	// 随机选择群用户
	rows, err = s.conn.QueryContext(ctx, sqlQueryGroupMember, groupID)
	if err != nil {
		return err
	}
	memberList := make([]int64, 0, 10)
	var memberID int64
	for rows.Next() {
		if err = rows.Scan(&memberID); err != nil {
			return err
		}
		memberList = append(memberList, memberID)
	}
	createHongbao := func() error {
		if _, err = s.conn.ExecContext(ctx, sqlUpdateUser, -amount, userID); err != nil {
			return err
		}
		res, err := s.conn.ExecContext(ctx, sqlInsertHongbao, userID, groupID, amount)
		if err != nil {
			return err
		}
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		remained := amount
		pickedNum := 0
		bestLuckUser := int64(0)
		pickupAmount := int64(0)
		maxAmount := int64(0)
		for _, id := range memberList {
			pickedNum++
			if pickedNum == len(memberList) {
				// 最后一人领取所有剩余红包金额
				pickupAmount = remained
				remained = 0
			} else {
				if remained > 1 {
					pickupAmount = 1 + rand.Int63n(remained-1)
				} else {
					pickupAmount = remained
				}
				remained -= pickupAmount
			}
			if pickupAmount > maxAmount {
				maxAmount = pickupAmount
				bestLuckUser = id
			}
			if _, err = s.conn.ExecContext(ctx, sqlUpdateUser, pickupAmount, id); err != nil {
				return err
			}
			if _, err = s.conn.ExecContext(ctx, sqlInsertHongbaoDetail, lastInsertID, id, pickupAmount); err != nil {
				return err
			}
			if amount == 0 {
				break
			}
		}
		if _, err = s.conn.ExecContext(ctx, sqlUpdateHongbao, bestLuckUser, maxAmount, pickedNum, lastInsertID); err != nil {
			return err
		}
		return nil
	}
	if err := s.BeginTxn(ctx); err != nil {
		return err
	}
	if err = createHongbao(); err != nil {
		if err2 := s.RollbackTxn(ctx); err2 != nil {
			log.Fatal(err2)
		}
		return err
	}
	if err = s.CommitTxn(ctx); err != nil {
		return err
	}
	// TODO: return details and lastInsertID
	return nil
}
