package writestress

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Table schema comes from the bank of pufa `tmp_jieb_instmnt_daily`
// CREATE TABLE `tmp_jieb_instmnt_daily` (
//   `ID` bigint(20) DEFAULT NULL COMMENT '主键ID',
//   `TABLE_ID` int(11) NOT NULL COMMENT '分库ID',
//   `FILE_DATE` char(8) NOT NULL COMMENT '文件日期',
//   `CONTRACT_NO` varchar(128) NOT NULL COMMENT '借据号',
//   `SETTLE_DATE` char(8) NOT NULL COMMENT '减免会计日期',
//   `TERM_NO` int(11) NOT NULL COMMENT '期次号',
//
//   `INPT_DATE` char(8) DEFAULT NULL COMMENT '录入日期',
//   `INPT_TIME` varchar(20) DEFAULT NULL COMMENT '录入时间',
//   `RCRD_ST_CODE` varchar(1) DEFAULT NULL COMMENT '记录状态代码',
//   UNIQUE KEY `TMP_JIEB_INSTMNT_DAILY_IDX1` (`CONTRACT_NO`,`TERM_NO`),
//   KEY `TMP_JIEB_INSTMNT_DAILY_IDX2` (`TABLE_ID`,`CONTRACT_NO`)
// ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin/*!90000 SHARD_ROW_ID_BITS=5 PRE_SPLIT_REGIONS=5 */ COMMENT='借呗日终（分期）信息临时表';
const (
	stmtDrop   = `DROP TABLE IF EXISTS write_stress`
	stmtCreate = `
	CREATE TABLE write_stress (
		TABLE_ID int(11) NOT NULL COMMENT '分库ID',
		CONTRACT_NO varchar(128) NOT NULL COMMENT '借据号',
		TERM_NO int(11) NOT NULL COMMENT '期次号',
		NOUSE char(255) NOT NULL COMMENT '填充位',
		
		UNIQUE KEY TMP_JIEB_INSTMNT_DAILY_IDX1 (CONTRACT_NO, TERM_NO),
		KEY TMP_JIEB_INSTMNT_DAILY_IDX2 (TABLE_ID, CONTRACT_NO)
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
`
)

// Config is for writestressClient
type Config struct {
	DataNum     int `toml:"dataNum"`
	Concurrency int `toml:"concurrency"`
	Batch       int `toml:"batch"`
}

// ClientCreator creates writestressClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node types.ClientNode) core.Client {
	return &writestressClient{
		Config: l.Cfg,
	}
}

// ledgerClient simulates a complete record of financial transactions over the
// life of a bank (or other company).
type writestressClient struct {
	*Config
	db       *sql.DB
	timeUnix int64
	rnd      *rand.Rand
}

func (c *writestressClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := nodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)

	log.Infof("start to init...")
	c.db, err = util.OpenDB(dsn, c.Concurrency)
	if err != nil {
		return err
	}
	defer func() {
		log.Infof("init end...")
	}()

	if _, err := c.db.Exec(stmtDrop); err != nil {
		log.Fatalf("execute statement %s error %v", stmtDrop, err)
	}

	if _, err := c.db.Exec(stmtCreate); err != nil {
		log.Fatalf("execute statement %s error %v", stmtCreate, err)
	}

	return nil
}

func (c *writestressClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

func (c *writestressClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

func (c *writestressClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *writestressClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *writestressClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
	log.Infof("start to test...")
	defer func() {
		log.Infof("test end...")
	}()

	c.rnd = rand.New(rand.NewSource(time.Now().Unix()))

	var wg sync.WaitGroup
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := c.ExecuteInsert(c.db, i); err != nil {
				log.Fatalf("exec failed %v", err)
			}
		}(i)
	}

	wg.Wait()
	return nil
}

// ExecuteInsert is run case
func (c *writestressClient) ExecuteInsert(db *sql.DB, pos int) error {
	totalNum := c.DataNum * 10000
	num := totalNum / c.Concurrency
	str := make([]byte, 250)

	limit := 1000
	if num < 100 {
		limit = 1
	} else if num < 1000 {
		limit = 100
	}
	timeUnix := c.timeUnix + int64(pos*num/limit)
	nextTimeUnix := c.timeUnix + int64((pos+1)*num/limit)
	count := 0
	for i := 0; i < num/c.Batch; i++ {
		tx, err := db.Begin()
		if err != nil {
			return errors.Trace(err)
		}
		n := num*pos + i*c.Batch
		if n >= totalNum {
			break
		}
		query := fmt.Sprintf(`INSERT INTO write_stress (TABLE_ID, CONTRACT_NO, TERM_NO, NOUSE) VALUES `)
		for j := 0; j < c.Batch; j++ {
			n := num*pos + i*c.Batch + j
			if n >= totalNum {
				break
			}
			// "abcd" + timestamp + count
			contract_id := []byte("abcd")
			tm := time.Unix(timeUnix, 0)
			contract_id = append(contract_id, tm.String()...)
			contract_id = append(contract_id, strconv.Itoa(count)...)
			util.RandString(str, c.rnd)
			if j != 0 {
				query += ","
			}

			query += fmt.Sprintf(`(%v, "%v", %v, "%v")`, c.rnd.Uint32()%960+1, string(contract_id[:]), c.rnd.Uint32()%36+1, string(str[:]))

			count++
			if count%limit == 0 {
				if timeUnix+1 == nextTimeUnix {
					count++
				} else {
					timeUnix++
					count = 0
				}
			}
		}
		//fmt.Println(query)
		if _, err := tx.Exec(query); err != nil {
			return errors.Trace(err)
		}
		if err := tx.Commit(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
