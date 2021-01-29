package blockwriter

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

const blockWriterBatchSize = 20

var gcInterval = 6 * time.Hour

// ClientCreator creates BlockWriteClient
type ClientCreator struct {
	TableNum    int
	Concurrency int
}

// Create creates WriterClient
func (c ClientCreator) Create(node cluster.ClientNode) core.Client {
	client := &WriterClient{
		tableNum:    c.TableNum,
		concurrency: c.Concurrency,
	}
	return client
}

// WriterClient is for concurrent writing blocks.
type WriterClient struct {
	tableNum    int
	concurrency int
	bws         []*blockWriter
	db          *sql.DB
}

// SetUp sets up client
func (c *WriterClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	var err error
	log.Infof("[%s] start to set up...", c)
	node := clientNodes[idx]
	c.db, err = util.OpenDB(fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port), c.concurrency)
	if err != nil {
		return err
	}

	c.bws = make([]*blockWriter, c.concurrency)
	for i := 0; i < c.concurrency; i++ {
		c.bws[i] = c.newBlockWriter()
	}
	defer func() {
		log.Infof("[%s] set up end...", c)
	}()
	for i := 0; i < c.tableNum; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		var s string
		if i > 0 {
			s = fmt.Sprintf("%d", i)
		}
		util.MustExec(c.db, fmt.Sprintf("CREATE TABLE IF NOT EXISTS block_writer%s %s", s, `
	(
      block_id BIGINT NOT NULL,
      writer_id VARCHAR(64) NOT NULL,
      block_num BIGINT NOT NULL,
      raw_bytes BLOB NOT NULL,
      PRIMARY KEY (block_id, writer_id, block_num)
)`))
	}

	return errors.Trace(c.truncate(ctx, c.db))
}

// TearDown tears down client
func (c *WriterClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start starts test
func (c *WriterClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	var wg sync.WaitGroup
	var ticker = time.NewTicker(gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := c.truncate(ctx, c.db)
			if err != nil {
				log.Errorf("[%s] truncate table error %v", c, err)
			}
		}
		for i := 0; i < c.concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						return
					default:
					}
					err := c.bws[i].batchExecute(c.db, c.tableNum)
					if err != nil {
						log.Errorf("[%s] batch execute error %v", c, err)
					}
				}
			}(i)
		}
		wg.Wait()
	}
}

type blockWriter struct {
	minSize         int
	maxSize         int
	id              string
	blockCount      uint64
	rand            *rand.Rand
	blockDataBuffer []byte
	values          []string
	index           int
}

func (c *WriterClient) newBlockWriter() *blockWriter {
	source := rand.NewSource(time.Now().UnixNano())
	return &blockWriter{
		id:              uuid.New().String(),
		rand:            rand.New(source),
		blockCount:      0,
		minSize:         128,
		maxSize:         1024,
		blockDataBuffer: make([]byte, 1024),
		values:          make([]string, blockWriterBatchSize),
	}
}

// Insert blockWriterBatchSize values in one SQL.
func (bw *blockWriter) batchExecute(db *sql.DB, tableNum int) error {
	log.Infof("[%s] table %d batch execution", "block_writer", tableNum)
	// buffer values
	for i := 0; i < blockWriterBatchSize; i++ {
		blockID := bw.rand.Int63()
		blockData := bw.randomBlock()
		bw.blockCount++
		bw.values[i] = fmt.Sprintf("(%d,'%s',%d,'%s')", blockID, bw.id, bw.blockCount, blockData)
	}
	var (
		err   error
		index string
	)

	if bw.index > 0 {
		index = fmt.Sprintf("%d", bw.index)
	}
	_, err = db.Exec(
		fmt.Sprintf(
			"INSERT INTO block_writer%s (block_id, writer_id, block_num, raw_bytes) VALUES %s",
			index, strings.Join(bw.values, ",")),
	)

	if err != nil {
		return fmt.Errorf("[block writer] insert err %v", err)
	}
	bw.index = (bw.index + 1) % tableNum
	return nil
}

func (bw *blockWriter) randomBlock() []byte {
	blockSize := bw.rand.Intn(bw.maxSize-bw.minSize) + bw.minSize

	util.RandString(bw.blockDataBuffer, bw.rand)
	return bw.blockDataBuffer[:blockSize]
}

func (c *WriterClient) truncate(ctx context.Context, db *sql.DB) error {
	for i := 0; i < c.tableNum; i++ {
		select {
		case <-ctx.Done():
			log.Errorf("[%s] truncate block write ctx done", c)
			return nil
		default:
		}
		var s string
		if i > 0 {
			s = fmt.Sprintf("%d", i)
		}
		log.Infof("[%s] truncate table block_writer%s", c, s)
		err := util.RunWithRetry(ctx, 10, 3*time.Second, func() error {
			_, err := db.Exec(fmt.Sprintf("truncate table block_writer%s", s))
			return err
		})
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// String implements fmt.Stringer interface.
func (c *WriterClient) String() string {
	return "block-writer"
}
