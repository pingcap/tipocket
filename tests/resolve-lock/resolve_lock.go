package resolvelock

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	pd "github.com/pingcap/pd/v4/client"
	"github.com/pingcap/tidb/store/tikv"
	"github.com/pingcap/tidb/store/tikv/oracle"
	"github.com/pingcap/tidb/store/tikv/tikvrpc"
	"github.com/pingcap/tidb/tablecodec"
	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	httputil "github.com/pingcap/tipocket/pkg/util/http"
	"github.com/pingcap/tipocket/util"
)

// Config is for resolveLockClient
type Config struct {
	EnableGreenGC bool
	RegionCount   int
	LockPerRegion int
	Worker        int
}

// Normalize normalizes unexpected config
func (c *Config) Normalize() *Config {
	if c.RegionCount == 0 {
		c.RegionCount = 1000
	}
	if c.LockPerRegion == 0 {
		c.LockPerRegion = 10
	}
	if c.Worker == 0 {
		c.Worker = 10
	}
	return c
}

// CaseCreator creates resolveLockClient
type CaseCreator struct {
	Cfg *Config
}

// Create creates the resolveLockClient from the CaseCreator
func (l CaseCreator) Create(node types.ClientNode) core.Client {
	return &resolveLockClient{
		Config: l.Cfg.Normalize(),
		dbName: "resolve_lock",
	}
}

type resolveLockClient struct {
	*Config

	dbName   string
	tableIDs []int64
	handleID int64

	safePoint  uint64
	safeLockTs uint64
	mockLockTs uint64

	dbStatusAddr string
	db           *sql.DB
	pd           pd.Client
	kv           tikv.Storage
}

func (c *resolveLockClient) openDB(ctx context.Context, ip string, port int32) error {
	dsn := fmt.Sprintf("root@tcp(%s:%d)/", ip, port)
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		return errors.Trace(err)
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", c.dbName))
	if err != nil {
		return errors.Trace(err)
	}
	db.Close()
	c.db, err = util.OpenDB(dsn+c.dbName, 100)
	return errors.Trace(err)
}

func (c *resolveLockClient) CreateTable(ctx context.Context, i int) (int64, error) {
	table := "t" + strconv.Itoa(i)
	_, err := c.db.ExecContext(ctx, fmt.Sprintf("create table if not exists %s(id int primary key, v varchar(128))", table))
	if err != nil {
		return 0, errors.Trace(err)
	}

	url := fmt.Sprintf("%s/schema/%s/%s", c.dbStatusAddr, c.dbName, table)
	resp, err := httputil.NewHTTPClient(http.DefaultClient).Get(url)
	if err != nil {
		return 0, errors.Trace(err)
	}
	var body struct {
		ID int64 `json:"id"`
	}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return body.ID, nil
}

func (c *resolveLockClient) SetUp(ctx context.Context, nodes []types.Node, clientNodes []types.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	log.Info("start to init")
	defer func() {
		log.Infof("init end")
	}()

	// PD
	pdNode := nodes[0]
	pdAddr := fmt.Sprintf("%s-pd.%s.svc:2379", pdNode.ClusterName, pdNode.Namespace)
	// NOTE: local run
	// pdAddr := "127.0.0.1:2379"
	pdClient, err := pd.NewClient([]string{pdAddr}, pd.SecurityOption{})
	if err != nil {
		return errors.Trace(err)
	}
	c.pd = pdClient

	// TiKV
	driver := tikv.Driver{}
	store, err := driver.Open(fmt.Sprintf("tikv://%s?disableGC=true", pdAddr))
	if err != nil {
		return errors.Trace(err)
	}
	c.kv = store.(tikv.Storage)

	// TiDB
	dbNode := clientNodes[idx]
	c.dbStatusAddr = fmt.Sprintf("http://%s-tidb.%s.svc:10080", dbNode.ClusterName, dbNode.Namespace)
	// NOTE: local run
	// c.dbStatusAddr = fmt.Sprintf("http://%s:10080", dbNode.IP)

	err = c.openDB(ctx, dbNode.IP, dbNode.Port)
	if err != nil {
		return errors.Trace(err)
	}
	// Disable GC
	_, err = c.db.ExecContext(ctx, `update mysql.tidb set VARIABLE_VALUE = "10000h" where VARIABLE_NAME in ("tikv_gc_run_interval", "tikv_gc_life_time")`)
	if err != nil {
		return errors.Trace(err)
	}
	log.Infof("create %d tables", c.RegionCount)
	// Can't create tables concurrently because there are too many WriteConflicts.
	for i := 0; i < c.RegionCount; i++ {
		id, err := c.CreateTable(ctx, i)
		if err != nil {
			return errors.Trace(err)
		}
		c.tableIDs = append(c.tableIDs, id)
	}

	return nil
}

func (c *resolveLockClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	c.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", c.dbName))
	return c.db.Close()
}

func (c *resolveLockClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

func (c *resolveLockClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *resolveLockClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *resolveLockClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
	log.Info("start to test")
	defer func() {
		log.Info("test end")
	}()

	lastGreenGC := -1
	for loopNum := 0; ; loopNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ts, err := c.getTs(ctx)
		if err != nil {
			return err
		}
		log.Infof("[round-%d] start to generate locks at ts(%v)", loopNum, ts)
		err = c.generateLocks(ctx, time.Microsecond)
		if err != nil {
			return errors.Trace(err)
		}

		// Sleep to let locks are applied in all replicas.
		time.Sleep(5 * time.Second)
		log.Infof("[round-%d] start to async generate locks during GC", loopNum)
		// Generate locks before ts to let lock observer do it job. ts is the safeLockTs which means
		// locks with ts before it are safe locks. These locks can be left after GC and won't break data consistency.
		cancel, wg := c.asyncGenerateLocksDuringGC(ctx, ts, 200*time.Millisecond, 2*time.Second)

		// Get a ts as the safe point so it's greater than any locks written by `generateLocks`
		c.safePoint, err = c.getTs(ctx)
		if err != nil {
			return errors.Trace(err)
		}
		log.Infof("[round-%d] start to GC at safePoint(%v)", loopNum, c.safePoint)
		// Invoke GC with the safe point
		greenGCUsed, err := c.resolveLocks(ctx)
		if err != nil {
			log.Errorf("[round-%d] failed to run GC at safe point %v", loopNum, c.safePoint)
			return errors.Trace(err)
		}
		log.Infof("[round-%d] GC done at safePoint(%v)", loopNum, c.safePoint)

		if greenGCUsed {
			lastGreenGC = loopNum
		} else {
			log.Warnf("[round-%d] failed to resolve lock physically at safe point %v", loopNum, c.safePoint)
		}
		if c.EnableGreenGC && loopNum-lastGreenGC > 50 {
			return errors.New("green gc failed to run for over 50 times")
		}

		log.Infof("[round-%d] start to check data at safePoint(%v)", loopNum, c.safePoint)
		// Cancel all goroutines that are generating locks asynchronously.
		cancel()
		wg.Wait()
		// Check there is no lock between safeLockTs and safePoint
		unsafeLocks, err := c.CheckData(ctx)
		if len(unsafeLocks) != 0 {
			log.Errorf("[round-%d] find %d unsafe locks after GC at safepoint(%v): %v", loopNum, len(unsafeLocks), c.safePoint, unsafeLocks)
			return errors.New("green GC check data failed")
		}
		if err != nil {
			return errors.Trace(err)
		}
		log.Infof("[round-%d] check data done at safePoint(%v)", loopNum, c.safePoint)
		c.reset(ctx)
	}
}

func (c *resolveLockClient) resolveLocks(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/test/gc/resolvelock?safepoint=%v&physical=%v", c.dbStatusAddr, c.safePoint, c.EnableGreenGC)
	resp, err := httputil.NewHTTPClient(http.DefaultClient).Get(url)
	if err != nil {
		return false, errors.Trace(err)
	}
	var body struct {
		PhysicalUsed bool `json:"physicalUsed"`
	}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return false, errors.Trace(err)
	}
	return body.PhysicalUsed, nil
}

func (c *resolveLockClient) asyncGenerateLocksDuringGC(ctx context.Context, safeLockTs uint64, interval time.Duration, timeout time.Duration) (context.CancelFunc, *sync.WaitGroup) {
	// Don't conflict with existing locks.
	c.handleID = int64(c.LockPerRegion)
	c.safeLockTs = safeLockTs
	c.mockLockTs = safeLockTs
	ctx, cancel := context.WithTimeout(ctx, timeout)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.generateLocks(ctx, interval)
	}()
	return cancel, &wg
}

func (c *resolveLockClient) generateLocks(ctx context.Context, interval time.Duration) error {
	type task struct {
		tableID  int64
		handleID int64
		limit    int
	}

	workers := c.Worker
	taskCh := make(chan task, len(c.tableIDs))
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for task := range taskCh {
				err := c.lock(ctx, task.tableID, task.handleID, task.limit)
				if err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}

	var err error
	ticker := time.NewTicker(interval)
	for _, tableID := range c.tableIDs {
		select {
		case <-ctx.Done():
			break
		case err = <-errCh:
			workers--
			break
		case <-ticker.C:
			taskCh <- task{tableID: tableID, handleID: c.handleID, limit: c.LockPerRegion}
		}
	}

	close(taskCh)
	for i := 0; i < workers; i++ {
		e := <-errCh
		if err == nil {
			err = e
		}
	}
	return err
}

func (c *resolveLockClient) lock(ctx context.Context, tableID int64, handleID int64, limit int) error {
	const txnSize = 5

	keys := make([][]byte, 0, txnSize)
	for i := 0; i < limit; i++ {
		keys = append(keys, tablecodec.EncodeRowKeyWithHandle(tableID, handleID+int64(i)))
		if len(keys) >= txnSize || i == limit-1 {
			_, err := c.lockBatch(ctx, keys, keys[0])
			if err != nil {
				return errors.Trace(err)
			}
			keys = keys[:0]
		}
	}
	return nil
}

func (c *resolveLockClient) lockBatch(ctx context.Context, keys [][]byte, primary []byte) (int, error) {
	const maxBatchSize = 16 * 1024

	// TiKV client doesn't expose Prewrite interface directly. We need to manually locate the region and send the
	// Prewrite requests.
	for {
		bo := tikv.NewBackoffer(ctx, 60000)
		loc, err := c.kv.GetRegionCache().LocateKey(bo, keys[0])
		if err != nil {
			return 0, errors.Trace(err)
		}

		// Get a timestamp to use as the startTs
		startTs, err := c.getLockTs(ctx)
		if err != nil {
			return 0, errors.Trace(err)
		}

		// Pick a batch of keys and make up the mutations
		batchSize := 0
		var mutations []*kvrpcpb.Mutation
		for _, key := range keys {
			if !loc.Contains(key) {
				break
			}
			value := []byte{'v'}
			mutations = append(mutations, &kvrpcpb.Mutation{
				Op:    kvrpcpb.Op_Put,
				Key:   key,
				Value: value,
			})

			batchSize += len(key) + len(value)
			if batchSize >= maxBatchSize {
				break
			}
		}

		lockedKeys := len(mutations)
		if lockedKeys == 0 {
			return 0, nil
		}

		req := tikvrpc.NewRequest(
			tikvrpc.CmdPrewrite,
			&kvrpcpb.PrewriteRequest{
				Mutations:    mutations,
				PrimaryLock:  primary,
				StartVersion: startTs,
				LockTtl:      30000,
			},
		)

		// Send the requests
		resp, err := c.kv.SendReq(bo, req, loc.Region, time.Second*20)
		if err != nil {
			return 0, errors.Annotatef(err, "send request failed. region: %+v [%+q, %+q), keys: %+q", loc.Region, loc.StartKey, loc.EndKey, keys[0:lockedKeys])
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return 0, errors.Trace(err)
		}
		if regionErr != nil {
			err = bo.Backoff(tikv.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return 0, errors.Trace(err)
			}
			continue
		}
		if resp.Resp == nil {
			return 0, errors.Errorf("response body missing")
		}
		// prewriteResp := resp.Resp.(*kvrpcpb.PrewriteResponse)
		// keyErrors := prewriteResp.GetErrors()
		// if len(keyErrors) != 0 {
		// return 0, errors.New(fmt.Sprintf("fail to prewrite locks: %v", keyErrors))
		// }

		return lockedKeys, nil
	}
}

func (c *resolveLockClient) CheckData(ctx context.Context) ([]*tikv.Lock, error) {
	const scanLockLimit = 100

	req := tikvrpc.NewRequest(tikvrpc.CmdScanLock, &kvrpcpb.ScanLockRequest{
		Limit:      10,
		MaxVersion: c.safePoint,
	})

	var unsafeLocks []*tikv.Lock
	key := make([]byte, 0)
	for {
		bo := tikv.NewBackoffer(ctx, 60000)

		req.ScanLock().StartKey = key
		loc, err := c.kv.GetRegionCache().LocateKey(bo, key)
		if err != nil {
			return unsafeLocks, errors.Trace(err)
		}
		resp, err := c.kv.SendReq(bo, req, loc.Region, 60*time.Second)
		if err != nil {
			return unsafeLocks, errors.Trace(err)
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return unsafeLocks, errors.Trace(err)
		}
		if regionErr != nil {
			err = bo.Backoff(tikv.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return unsafeLocks, errors.Trace(err)
			}
			continue
		}
		if resp.Resp == nil {
			return unsafeLocks, errors.New("missing response body")
		}
		scanLockResp := resp.Resp.(*kvrpcpb.ScanLockResponse)
		if scanLockResp.GetError() != nil {
			return unsafeLocks, errors.Errorf("unexpected scanlock error: %s", scanLockResp)
		}

		locksInfo := scanLockResp.GetLocks()
		safeLocks := make([]*tikv.Lock, 0, len(locksInfo))
		for _, info := range locksInfo {
			lock := tikv.NewLock(info)
			if lock.TxnID < c.safeLockTs {
				safeLocks = append(safeLocks, lock)
			} else {
				unsafeLocks = append(unsafeLocks, lock)
			}
		}
		if len(safeLocks) != 0 {
			log.Infof("found %d locks after GC at safePoint(%v)", len(safeLocks), c.safePoint)
		}

		ok, err := c.kv.GetLockResolver().BatchResolveLocks(bo, safeLocks, loc.Region)
		if err != nil {
			return unsafeLocks, errors.Trace(err)
		}
		if !ok {
			err = bo.Backoff(tikv.BoTxnLock, errors.Errorf("remain locks: %d", len(safeLocks)))
			if err != nil {
				return unsafeLocks, errors.Trace(err)
			}
			continue
		}
		if len(locksInfo) < scanLockLimit {
			key = loc.EndKey
		} else {
			key = locksInfo[len(locksInfo)-1].GetKey()
		}
		if len(key) == 0 {
			break
		}
	}
	return unsafeLocks, nil
}

func (c *resolveLockClient) reset(ctx context.Context) {
	c.handleID = 0
	c.safePoint = 0
	c.safeLockTs = 0
	c.mockLockTs = 0
}

func (c *resolveLockClient) getTs(ctx context.Context) (uint64, error) {
	physical, logical, err := c.pd.GetTS(ctx)
	if err != nil {
		return 0, errors.Trace(err)
	}
	ts := oracle.ComposeTS(physical, logical)
	return ts, nil
}

func (c *resolveLockClient) getLockTs(ctx context.Context) (uint64, error) {
	if c.mockLockTs == 0 {
		return c.getTs(ctx)
	}
	c.mockLockTs--
	return c.mockLockTs, nil
}
