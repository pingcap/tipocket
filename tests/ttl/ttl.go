package ttl

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngaut/log"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

var (
	typicalTTLs = [...]uint64{
		0,
		600,  // 10m
		3600, // 1h
	}
	zeroTTLVerifyTime uint64 = 600
	toleranceOfTTL    uint64 = 60
	placeHolderValue         = [...]byte{42, 42}
)

// ClientCreator ...
type ClientCreator struct {
	Cfg *Config
}

// Config ...
type Config struct {
	Concurrency   int
	DataPerWorker int
}

type ttlClient struct {
	cfg     *Config
	cli     *rawkv.Client
	wg      sync.WaitGroup
	stopped int32
}

func (c *ttlClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	log.Infof("setup...")

	clusterName := clientNodes[0].ClusterName
	ns := clientNodes[0].Namespace
	pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}
	if len(pdAddrs) == 0 {
		return errors.New("no pd node found")
	}

	var err error
	c.cli, err = rawkv.NewClient(ctx, pdAddrs, config.Default())
	if err != nil {
		log.Fatalf("create tikv client error: %v", err)
	}

	log.Infof("setup client %v end", idx)
	return nil
}

func (c *ttlClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start implements Case Start interface.
func (c *ttlClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	var wg sync.WaitGroup

	run := func(id int, f func(id int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if atomic.LoadInt32(&c.stopped) != 0 {
					log.Errorf("[%s] case stopped", c)
					return
				}
				c.wg.Add(1)
				f(id)
				c.wg.Done()
			}
		}()
	}

	for i := 0; i < c.cfg.Concurrency; i++ {
		run(i, func(id int) {
			randIdx := rand.Intn(len(typicalTTLs))
			TTL := typicalTTLs[randIdx]
			prefix := c.intToBigEndianByte(i)
			log.Infof("[%s] run case, id %i, TTL %s", c, i, TTL)
			c.RunCase(ctx, prefix, TTL)
			log.Infof("[%s] case end, id %i, TTL %s", c, i, TTL)
		})
	}
	return nil
}

// RunCase run test case with a key prefix.
// Key => [8 byte of prefix | 8 byte of in-case id]
func (c *ttlClient) RunCase(ctx context.Context, prefix []byte, TTL uint64) {
	var keys, values [][]byte
	for i := c.cfg.DataPerWorker; i < c.cfg.DataPerWorker; i++ {
		keys = append(keys, c.keyFromID(prefix, i))
		values = append(values, placeHolderValue[:])
	}
	if len(keys) == 0 {
		log.Infof("[%s] no valid keys")
		return
	}
	log.Infof("[%s] generated key range, startkey %s, endkey %s", c, keys[0], keys[len(keys)-1])

	// before inserting any data ttl should be nil.
	c.checkTTLNil(ctx, keys)

	// Test `Put` with half of data, `BatchPut` with another half
	half := len(keys) / 2
	c.seperatePutWithTTL(ctx, keys[:half], values[:half], TTL)
	c.batchPutWithTTL(ctx, keys[half:], values[half:], TTL)

	var sleepTime time.Duration
	var expectTTL uint64
	if TTL == 0 {
		sleepTime = time.Duration(zeroTTLVerifyTime) * time.Second
		expectTTL = 0
	} else {
		sleepTime = time.Duration(TTL/2) * time.Second
		expectTTL = TTL / 2
	}

	time.Sleep(sleepTime)
	if atomic.LoadInt32(&c.stopped) != 0 {
		return
	}

	// Test Get
	c.expectGetSucceed(ctx, keys, values, TTL)
	c.expectKeyTTL(ctx, keys, TTL, expectTTL, toleranceOfTTL)

	// Test Scan
	c.expectScanSucceed(ctx, keys, values, c.cfg.DataPerWorker+1, TTL)

	// Random pick a non zero TTL
	for {
		randIdx := rand.Intn(len(typicalTTLs))
		TTL = typicalTTLs[randIdx]
		if TTL != 0 {
			break
		}
	}
	// Update TTL.
	half = len(keys) / 2
	c.batchPutWithTTL(ctx, keys[half:], values[half:], TTL)
	c.seperatePutWithTTL(ctx, keys[:half], values[:half], TTL)
	c.expectKeyTTL(ctx, keys, TTL, TTL, toleranceOfTTL)

	sleepTime = time.Duration(TTL/2) * time.Second
	expectTTL = TTL / 2
	time.Sleep(sleepTime)

	// Test Get
	c.expectGetSucceed(ctx, keys, values, TTL)
	c.expectKeyTTL(ctx, keys, TTL, expectTTL, toleranceOfTTL)

	// Test Scan
	c.expectScanSucceed(ctx, keys, values, c.cfg.DataPerWorker+1, TTL)

	// Sleep until TTL expired.
	// Total sleep time = TTL + 4 * tolerance
	sleepTime = time.Duration(TTL/2+4*toleranceOfTTL) * time.Second
	time.Sleep(sleepTime)
	c.expectGetNilValue(ctx, keys)
}

// Any error in checkTTLNil will cause case to fail.
func (c *ttlClient) checkTTLNil(ctx context.Context, keys [][]byte) {
	for _, key := range keys {
		ttl, err := c.cli.GetKeyTTL(ctx, key)
		if err != nil {
			// TODO: Should this fatal?
			log.Infof("[%s] verify error: %s in %s", c, err, time.Now())
		}
		if ttl != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] ttl of key %s exists before insertion, ttl value %s", c, key, ttl))
		}
	}
}

// Any error in seperatePutWithTTL will cause case to fail.
func (c *ttlClient) seperatePutWithTTL(ctx context.Context, keys, values [][]byte, TTL uint64) {
	for i, key := range keys {
		err := c.cli.Put(
			ctx,
			key,
			values[i],
			rawkv.PutOption{TTL: TTL},
		)
		if err != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL put error %s, key %s, TTL %s seconds", c, err, key, TTL))
		}
	}
}

// Any error in batchPutWithTTL will cause case to fail.
func (c *ttlClient) batchPutWithTTL(ctx context.Context, keys, values [][]byte, TTL uint64) {
	err := c.cli.BatchPut(
		ctx,
		keys,
		values,
		rawkv.PutOption{TTL: TTL},
	)
	if err != nil {
		c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL batch put error %s, TTL %s seconds", c, err, TTL))
	}
}

func (c *ttlClient) expectGetSucceed(ctx context.Context, keys, values [][]byte, TTL uint64) {
	for i, key := range keys {
		val, err := c.cli.Get(ctx, key)
		if err != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL get error %s, key %s, TTL %s seconds", c, err, key, TTL))
		}
		if bytes.Equal(val, values[i]) {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL get value error, on key %s, expect %s, get %s, TTL %s seconds", c, err, key, values[i], TTL))
		}
	}
}


func (c *ttlClient) expectGetNilValue(ctx context.Context, keys [][]byte) {
	for _, key := range keys {
		val, err := c.cli.Get(ctx, key)
		if err != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL expect get failed error %s, key %s", c, err, key))
		}
		if val != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL get unexpected value, key %s, val %s", c, key, val))
		}
	}
}

func (c *ttlClient) expectKeyTTL(ctx context.Context, keys [][]byte, TTL, expectTTL uint64, tolerance uint64) {
	var eqInTolerance func(uint64, uint64) bool = equalInToleranceCreator(tolerance)
	for _, key := range keys {
		ttl, err := c.cli.GetKeyTTL(ctx, key)
		if err != nil {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL GetKeyTTL error %s, key %s, TTL %s seconds", c, err, key, TTL))
		}
		if !eqInTolerance(*ttl, expectTTL) {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL time error, get ttl %s, expect ttl %s", c, *ttl, expectTTL))
		}
	}
}

func (c *ttlClient) expectScanSucceed(ctx context.Context, keys, values [][]byte, limit int, TTL uint64) {
	// TODO: A better scan test method is required...
	startKey := keys[0]
	endKey := append(keys[len(keys)-1], '\0')
	scanKeys, scanValues, err := c.cli.Scan(
		ctx,
		startKey,
		endKey,
		limit,
	)
	if err != nil {
		c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL scan error %s, startkey %s, endkey %s, TTL %s seconds", c, err, startKey, endKey, TTL))
	}
	for i, key := range scanKeys {
		if !bytes.Equal(values[i], scanValues[i]) {
			c.waitAndFatal(fmt.Sprintf("[%s] RawKV TTL scan value error %s, on key %s, expect %s, get %s, TTL %s seconds", c, err, key, values[i], scanValues[i], TTL))
		}
	}
}

func (c *ttlClient) waitAndFatal(errmsg string) {
	log.Errorf(errmsg)
	atomic.StoreInt32(&c.stopped, 1)
	c.wg.Wait()
	log.Fatalf(errmsg)
}

func (c *ttlClient) keyFromID(prefix []byte, dataID int) []byte {
	key := prefix // Copy prefix
	key = append(key, c.intToBigEndianByte(dataID)...)
	return key
}

// intToBigEndianByte returns a byte array, persist the order of i.
func (c *ttlClient) intToBigEndianByte(i int) []byte {
	ans := make([]byte, 8)
	iU64 := uint64(i)
	binary.BigEndian.PutUint64(ans, iU64)
	return ans
}

func (c *ttlClient) String() string {
	return "ttl"
}

// equalInToleranceCreator create a function tests if abs(lhs - rhs) <= tolerance.
func equalInToleranceCreator(tolerance uint64) func(uint64, uint64) bool {
	return func(lhs, rhs uint64) bool {
		gap := lhs - rhs
		if gap < 0 {
			gap = -gap
		}
		if gap <= tolerance {
			return true
		} else {
			return false
		}
	}
}

// Create ...
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &ttlClient{
		cfg: c.Cfg,
	}
}
