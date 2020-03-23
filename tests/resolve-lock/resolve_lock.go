package resolvelock

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/pingcap/tidb/store/tikv/gcworker"
	"math/rand"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	pd "github.com/pingcap/pd/v4/client"
	"github.com/pingcap/tidb/store/tikv"
	"github.com/pingcap/tidb/store/tikv/oracle"
	"github.com/pingcap/tidb/store/tikv/tikvrpc"
	"github.com/pingcap/tidb/util/codec"
	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config is for resolveLockClient
type Config struct {
	Concurrency             int
	EnableGreenGC           bool
	TableSize               int
	LocksPerRegion          int
	GenerateLockConcurrency int
}

// CaseCreator creates resolveLockClient
type CaseCreator struct {
	Cfg *Config
}

// Create creates the resolveLockClient from the CaseCreator
func (l CaseCreator) Create(node types.ClientNode) core.Client {
	return &resolveLockClient{
		Config: l.Cfg,
	}
}

type resolveLockClient struct {
	*Config
	db *sql.DB

	pd pd.Client
	kv tikv.Storage
}

func (c *resolveLockClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
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

	// Disable GC
	_, err = c.db.Exec(`update mysql.tidb set VARIABLE_VALUE = "10000h" where VARIABLE_NAME in ("tikv_gc_run_interval", "tikv_gc_life_time")`)
	if err != nil {
		return errors.Trace(err)
	}

	// We are not going to let TiDB manage the GC process. So this config is actually useless.
	//scanLockMode := "legacy"
	//if c.EnableGreenGC {
	//	scanLockMode = "physical"
	//}
	//_, err = c.db.Exec(`insert into mysql.tidb values ('tikv_gc_scan_lock_mode', ?, '')
	//		       on duplicate key
	//		       update variable_value = ?, comment = ''`, scanLockMode, scanLockMode)
	//if err != nil {
	//	return errors.Trace(err)
	//}

	if c.TableSize == 0 {
		log.Infof("table size is set to 0. Do not load data.")
		return nil
	}
	_, err = c.db.Exec("create table t1(id integer, v varchar(128), primary key (id))")
	if err != nil {
		return errors.Trace(err)
	}

	const loadDataConcurrency = 100

	errCh := make(chan error, loadDataConcurrency)
	for threadNum := 0; threadNum < loadDataConcurrency; threadNum++ {
		threadNum1 := threadNum
		go func() {
			for i := threadNum1; i < c.TableSize; i += loadDataConcurrency {
				_, err := c.db.Exec("insert into t1 values(?, ?)", i, randStr())
				if err != nil {
					errCh <- errors.Trace(err)
					return
				}
			}
			errCh <- nil
		}()
	}

	for i := 0; i < loadDataConcurrency; i++ {
		err = <-errCh
		if err != nil {
			return errors.Trace(err)
		}
	}

	clusterName := nodes[0].ClusterName
	ns := nodes[0].Namespace
	pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}

	if len(pdAddrs) == 0 {
		return errors.New("No pd node found")
	}

	// TODO: Is SecurityOption needed?
	c.pd, err = pd.NewClient(pdAddrs, pd.SecurityOption{})
	driver := tikv.Driver{}
	store, err := driver.Open(fmt.Sprintf("tikv://%s?disableGC=true", strings.Join(pdAddrs, ",")))
	c.kv = store.(tikv.Storage)

	return nil
}

func (c *resolveLockClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

func (c *resolveLockClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) interface{} {
	panic("implement me")
}

func (c *resolveLockClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *resolveLockClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *resolveLockClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
	log.Infof("start to test...")
	defer func() {
		log.Infof("test end...")
	}()

	lastGreenGC := -1

	for loopNum := 0; ; loopNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := c.generateLocks(ctx)
		if err != nil {
			return errors.Trace(err)
		}

		// Get a ts as the safe point so it's greater than any locks written by `generateLocks`
		safePoint, err := c.getTs(ctx)
		if err != nil {
			return errors.Trace(err)
		}

		// Invoke GC with the safe point
		id := fmt.Sprintf("gc-worker-%v", loopNum)
		greenGCUsed, err := gcworker.RunDistributedGCJob(ctx, c.kv, c.pd, safePoint, id, 3, c.EnableGreenGC)
		if err != nil {
			return errors.Trace(err)
		}

		if greenGCUsed {
			lastGreenGC = loopNum
		}

		if c.EnableGreenGC && loopNum-lastGreenGC > 10 {
			return errors.New("green gc failed to run for over 10 times")
		}

		// Check there is no lock before safePoint
		err = c.CheckData(ctx, safePoint)
		if err != nil {
			return errors.Trace(err)
		}
	}
}

// generateLocks sends Prewrite requests to TiKV to generate locks, without committing and rolling back.
func (c *resolveLockClient) generateLocks(ctx context.Context) error {
	log.Infof("Start generating lock")
	errCh := make(chan error, c.GenerateLockConcurrency)
	key := make([]byte, 0)
	lastEndKey := make([]byte, 0)
	currentRunningRegions := 0

	for {
		regions, _, err := c.pd.ScanRegions(ctx, key, []byte{}, 256)
		if err != nil {
			return errors.Trace(err)
		}

		nextScanKey := regions[len(regions)-1].EndKey
		nextScanKeyCopy := make([]byte, len(nextScanKey))
		copy(nextScanKeyCopy, nextScanKey)

		for _, region := range regions {
			err = decodeRegionMetaKey(region)
			if err != nil {
				return errors.Trace(err)
			}

			if len(region.EndKey) > 0 && bytes.Compare(region.EndKey, lastEndKey) <= 0 {
				continue
			}
			if bytes.Compare(region.StartKey, lastEndKey) < 0 {
				region.StartKey = lastEndKey
			}
			lastEndKey = region.EndKey

			for currentRunningRegions >= c.GenerateLockConcurrency {
				select {
				case <-ctx.Done():
					return nil
				case err = <-errCh:
					if err != nil {
						return errors.Trace(err)
					}
					currentRunningRegions--
				}
			}
			currentRunningRegions++
			go func() {
				// Retry for several times to reduce the possibility of failing when the cluster is unstable
				var err error
				for retry := 0; retry < 10; retry++ {
					err = c.singleLockRegion(ctx, region.StartKey, region.EndKey)
					if err == nil {
						errCh <- nil
						return
					}
				}
				errCh <- err
			}()
		}

		key = nextScanKeyCopy
		if len(key) == 0 {
			break
		}
	}

	for currentRunningRegions >= c.GenerateLockConcurrency {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err != nil {
				return errors.Trace(err)
			}
			currentRunningRegions--
		}
	}

	return nil
}

func (c *resolveLockClient) singleLockRegion(ctx context.Context, startKey, endKey []byte) error {
	ts, err := c.kv.CurrentVersion()
	if err != nil {
		return errors.Trace(err)
	}
	snap, err := c.kv.GetSnapshot(ts)
	if err != nil {
		return errors.Trace(err)
	}
	iter, err := snap.Iter(startKey, endKey)
	if err != nil {
		return errors.Trace(err)
	}

	if !iter.Valid() {
		return nil
	}

	err = c.lockKeys(ctx, [][]byte{iter.Key()})
	return errors.Trace(err)
}

func (c *resolveLockClient) lockKeys(ctx context.Context, keys [][]byte) error {
	primary := keys[0]

	for len(keys) > 0 {
		lockedKeys, err := c.lockBatch(ctx, keys, primary)
		if err != nil {
			return errors.Trace(err)
		}
		keys = keys[lockedKeys:]
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
		startTs, err := c.getTs(ctx)
		if err != nil {
			return 0, errors.Trace(err)
		}

		// Pick a batch of keys and make up the mutations
		var mutations []*kvrpcpb.Mutation
		batchSize := 0

		for _, key := range keys {
			if len(loc.EndKey) > 0 && bytes.Compare(key, loc.EndKey) >= 0 {
				break
			}
			if bytes.Compare(key, loc.StartKey) < 0 {
				break
			}

			value := randStr()
			mutations = append(mutations, &kvrpcpb.Mutation{
				Op:    kvrpcpb.Op_Put,
				Key:   key,
				Value: []byte(value),
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
				LockTtl:      1000 * 60,
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

		prewriteResp := resp.Resp.(*kvrpcpb.PrewriteResponse)
		if prewriteResp == nil {
			return 0, errors.Errorf("response body missing")
		}

		// Ignore key errors since we never commit the transaction and we don't need to keep consistency here.
		return lockedKeys, nil
	}
}

func (c *resolveLockClient) CheckData(ctx context.Context, safePoint uint64) error {
	nextKey := make([]byte, 0)

	for {
		req := tikvrpc.NewRequest(
			tikvrpc.CmdScanLock,
			&kvrpcpb.ScanLockRequest{
				StartKey:   nextKey,
				Limit:      10,
				MaxVersion: safePoint,
			},
		)
		bo := tikv.NewBackoffer(ctx, 60000)
		loc, err := c.kv.GetRegionCache().LocateKey(bo, nextKey)
		if err != nil {
			return errors.Trace(err)
		}

		resp, err := c.kv.SendReq(bo, req, loc.Region, 60*time.Second)
		if err != nil {
			return errors.Trace(err)
		}
		regionErr, err := resp.GetRegionError()
		if err != nil {
			return errors.Trace(err)
		}
		if regionErr != nil {
			err = bo.Backoff(tikv.BoRegionMiss, errors.New(regionErr.String()))
			if err != nil {
				return errors.Trace(err)
			}
			continue
		}
		scanLockResp := resp.Resp.(*kvrpcpb.ScanLockResponse)
		if scanLockResp == nil {
			return errors.New("missing response body")
		}

		if len(scanLockResp.Locks) > 0 {
			return errors.Errorf("check data failed: lock before safePoint is not empty. locks: %+q", scanLockResp.Locks)
		}

		nextKey = loc.EndKey
		if len(nextKey) == 0 {
			// Finished scanning the whole store.
			log.Infof("Check data success on safePoint %v.", safePoint)
			return nil
		}
	}
}

func (c *resolveLockClient) getTs(ctx context.Context) (uint64, error) {
	physical, logical, err := c.pd.GetTS(ctx)
	if err != nil {
		return 0, errors.Trace(err)
	}
	ts := oracle.ComposeTS(physical, logical)
	return ts, nil
}

func randStr() string {
	length := rand.Intn(128)
	res := ""
	for i := 0; i < length; i++ {
		res += string('a' + (rand.Int() % 26))
	}
	return res
}

func decodeRegionMetaKey(r *metapb.Region) error {
	if len(r.StartKey) != 0 {
		_, decoded, err := codec.DecodeBytes(r.StartKey, nil)
		if err != nil {
			return errors.Trace(err)
		}
		r.StartKey = decoded
	}
	if len(r.EndKey) != 0 {
		_, decoded, err := codec.DecodeBytes(r.EndKey, nil)
		if err != nil {
			return errors.Trace(err)
		}
		r.EndKey = decoded
	}
	return nil
}
