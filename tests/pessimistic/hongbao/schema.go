package hongbao

import (
	"context"
	"database/sql"
)

const createUserTable = `
create table user(
  uid int not null AUTO_INCREMENT COMMENT 'PK',
  uname VARCHAR(20) not null  COMMENT '名字',
  birth_day date not null COMMENT '出生',
  addr_province int not null COMMENT '省份',
  addr_city int not null COMMENT '城市',
  friends int not null COMMENT '朋友数',
  balance int not null default 0 COMMENT '用户余额,单位为分',
  add_time datetime not null default CURRENT_TIMESTAMP COMMENT '新增时间',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(uid),
  unique key(uname),
  key(uid,balance),
  key(addr_city),
  key(upt_time)
)engine=innodb, COMMENT '用户表';
`

const createUserBankTable = `
create table user_bank(
  uid int not null COMMENT '用户id',
  balance int not null default 100000000  COMMENT '用户余额,单位为分',
  add_time datetime not null default CURRENT_TIMESTAMP COMMENT '新增时间',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(uid),
  key(upt_time)
)engine=innodb, COMMENT '用户银存款表';
`

const createUserFriendsTable = `
create table user_friends(
  uid int not null COMMENT '用户id',
  ufid int not null  COMMENT '朋友id',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(uid,ufid),
  unique key(ufid,uid),
  key(upt_time)
)engine=innodb, COMMENT '用户朋友表';
`

const createUserBillIn = `
create table user_bill_in(
  id int not null COMMENT 'PK',
  uid int not null COMMENT '用户id',
  bill_type TINYINT not null COMMENT '帐单类型: 1-冲值,2-红包',
  bill_datetime datetime not null default CURRENT_TIMESTAMP COMMENT '交易时间',
  bill_amount int not null COMMENT '交易金额',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(id),
  key(upt_time)
)engine=innodb, COMMENT '用户冲值记录表';
`

const createUserBillOut = `
create table user_bill_out(
  id int not null COMMENT 'PK',
  uid int not null COMMENT '用户id',
  bill_type TINYINT not null COMMENT '帐单类型:1-提现 2-红包',
  bill_datetime datetime not null default CURRENT_TIMESTAMP COMMENT '交易时间',
  bill_amount int not null COMMENT '交易金额',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(id),
  key(upt_time)
)engine=innodb, COMMENT '用户消费记录表';
`

const createGroup = `
create table group_info(
  gid int not null AUTO_INCREMENT COMMENT '群id',
  create_uid int not null COMMENT '创建者',
  gname varchar(20) not null  COMMENT '群名称',
  group_members int not null DEFAULT 0 COMMENT '群成员总数',
  add_time datetime not null default CURRENT_TIMESTAMP COMMENT '新增时间',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(gid),
  unique key(gname),
  key(add_time),
  key(upt_time)
)engine=innodb, COMMENT '用户群表';
`

const createGroupMember = `
create table group_member(
  gid int not null COMMENT '群id',
  uid int not null COMMENT '用户id',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(gid,uid),
  unique key(uid,gid),
  key(upt_time)
)engine=innodb, COMMENT '群成员表';
`

const createHongbao = `
create table hongbao(
  reid int not null AUTO_INCREMENT COMMENT '红包id',
  uid int not null COMMENT '用户id',
  gid int not null COMMENT '群id',
  amount int not null COMMENT '红包金额',
  best_luck_uid int not null default 0 COMMENT '最佳手气用户id',
  max_mount int not null default 0 COMMENT '最大金额',
  pickup_users int not null default 0 COMMENT '领取人数',
  add_time datetime not null default CURRENT_TIMESTAMP COMMENT '新增时间',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(reid),
  key(upt_time),
  key(add_time)
)engine=innodb, COMMENT '红包表';
`
const createHongbaoDetail = `
create table hongbao_detail(
  id int not null AUTO_INCREMENT COMMENT 'PK',
  reid int not null COMMENT '红包id',
  uid int not null COMMENT '用户id',
  amount int not null COMMENT '领取金额',
  add_time datetime not null default CURRENT_TIMESTAMP COMMENT '领取时间',
  upt_time datetime not null default CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  primary key(id),
  key(reid),
  key(add_time),
  key(upt_time)
)engine=innodb, COMMENT '红包明细表';
`

var stmts = []string{
	createUserTable,
	createUserBankTable,
	createUserFriendsTable,
	createUserBillIn,
	createUserBillOut,
	createGroup,
	createGroupMember,
	createHongbao,
	createHongbaoDetail,
}

func InitSchema(ctx context.Context, db *sql.DB) error {
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
