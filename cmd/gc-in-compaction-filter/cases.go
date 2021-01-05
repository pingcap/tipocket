package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/kvproto/pkg/debugpb"
	tidbcodec "github.com/pingcap/tidb/util/codec"
	"github.com/tikv/client-go/rawkv"
	"google.golang.org/grpc"
)

const (
	timeFence = 1000 * 60 * 10 // A large gap(in ms) before safe point.
)

func (c *client) mustGetSafePoint() uint64 {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		resp, err := c.etcdCli.Get(ctx, "/tidb/store/gcworker/saved_safe_point")
		if err != nil {
			log.Fatalf("[%s] get safe point error %v", caseLabel, err)
		}
		if len(resp.Kvs) == 0 {
			log.Infof("[%s] no safe point in pd", caseLabel)
			time.Sleep(10 * time.Second)
			continue
		}
		value := string(resp.Kvs[0].Value)
		t, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			log.Fatalf("[%s] parse safe point error %v", caseLabel, err)
		}
		log.Infof("[%s] load safe point %d", caseLabel, t)
		return t
	}
}

func (c *client) mustGetTiKVCtlClient(kvAddr string) debugpb.DebugClient {
	u, err := url.Parse(fmt.Sprintf("http://%s", kvAddr))
	if err != nil {
		log.Fatalf("[%s] parse tikv addr error %v", caseLabel, err)
	}
	conn, err := grpc.DialContext(context.TODO(), u.Host, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("[%s] connect to tikv error %v", caseLabel, err)
	}
	return debugpb.NewDebugClient(conn)
}

func gcByCompact(debugCli debugpb.DebugClient) {
	req := &debugpb.CompactRequest{
		Db:      debugpb.DB_KV,
		Cf:      "write",
		Threads: 1,
		// It's the default value used in TiKV internal compactions.
		BottommostLevelCompaction: debugpb.BottommostLevelCompaction_Force,
	}
	if _, err := debugCli.Compact(context.TODO(), req); err != nil {
		log.Fatalf("[%s] compact tikv error %v", caseLabel, err)
	}
	time.Sleep(5 * time.Second)
}

func genMvccPut(key []byte, value []byte, startTs uint64, commitTs uint64) ([]byte, []byte) {
	writeKey := tidbcodec.EncodeUintDesc(key, commitTs)
	writeValue := make([]byte, 0, 256)
	writeValue = append(writeValue, 'P') // type
	writeValue = tidbcodec.EncodeUvarint(writeValue, startTs)
	writeValue = append(writeValue, 'v') // short value prefix
	writeValue = append(writeValue, byte(len(value)))
	writeValue = append(writeValue, value...)
	return writeKey, writeValue
}

func genMvccDelete(key []byte, startTs uint64, commitTs uint64) ([]byte, []byte) {
	writeKey := tidbcodec.EncodeUintDesc(key, commitTs)
	writeValue := make([]byte, 0, 256)
	writeValue = append(writeValue, 'D') // type
	writeValue = tidbcodec.EncodeUvarint(writeValue, startTs)
	return writeKey, writeValue
}

func genRandomBytes(size int) (blk []byte) {
	blk = make([]byte, size)
	_, _ = rand.Read(blk)
	return
}

// Case 1: test old versions before the safe point will be cleaned by compaction.
func (c *client) testGcStaleVersions() {
	var err error
	debugCli := c.mustGetTiKVCtlClient(c.kvAddrs[0])

	_, err = c.db.Exec("set config tikv gc.ratio-threshold = 0.9")
	if err != nil {
		log.Fatalf("[%s] change gc.ratio-threshold fail, error %v", caseLabel, err)
	}

	key := append([]byte("test_gc_stale_versions"), genRandomBytes(8)...)
	value := []byte("test_gc_stale_versions")

	safepoint := c.mustGetSafePoint()
	physical := safepoint >> 18
	oldStartTs := (physical - timeFence) << 18
	oldCommitTs := oldStartTs + 1000
	newStartTs := (physical - timeFence + 10) << 18
	newCommitTs := newStartTs + 1000

	oldKey, oldValue := genMvccPut(key, value, oldStartTs, oldCommitTs)
	err = c.rawKvCli.Put(context.TODO(), oldKey, oldValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	newKey, newValue := genMvccPut(key, value, newStartTs, newCommitTs)
	err = c.rawKvCli.Put(context.TODO(), newKey, newValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	gcByCompact(debugCli)
	value, err = c.rawKvCli.Get(context.TODO(), oldKey, rawkv.GetOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv get error %v", caseLabel, err)
	}
	if value != nil {
		log.Fatalf("[%s] stale version is not cleaned", caseLabel)
	}
}

// Case 2: test the latest version before the safe point won't be cleaned incorrectly.
func (c *client) testGcLatestPutBeforeSafePoint() {
	var err error
	debugCli := c.mustGetTiKVCtlClient(c.kvAddrs[0])

	_, err = c.db.Exec("set config tikv gc.ratio-threshold = 0.9")
	if err != nil {
		log.Fatalf("[%s] change gc.ratio-threshold fail, error %v", caseLabel, err)
	}

	key := append([]byte("test_gc_latest_put"), genRandomBytes(8)...)
	value := []byte("test_gc_latest_put")

	safepoint := c.mustGetSafePoint()
	physical := safepoint >> 18
	oldStartTs := (physical - timeFence) << 18
	oldCommitTs := oldStartTs + 1000
	newStartTs := (physical + timeFence) << 18
	newCommitTs := newStartTs + 1000

	oldKey, oldValue := genMvccPut(key, value, oldStartTs, oldCommitTs)
	err = c.rawKvCli.Put(context.TODO(), oldKey, oldValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	gcByCompact(debugCli)
	value, err = c.rawKvCli.Get(context.TODO(), oldKey, rawkv.GetOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv get error %v", caseLabel, err)
	}
	if !sliceEqual(value, oldValue) {
		log.Fatalf("[%s] the latest version before safe point is cleaned", caseLabel)
	}

	newKey, newValue := genMvccPut(key, value, newStartTs, newCommitTs)
	err = c.rawKvCli.Put(context.TODO(), newKey, newValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	gcByCompact(debugCli)
	value, err = c.rawKvCli.Get(context.TODO(), oldKey, rawkv.GetOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv get error %v", caseLabel, err)
	}
	if !sliceEqual(value, oldValue) {
		log.Fatalf("[%s] the latest version before safe point is cleaned", caseLabel)
	}
}

// Case 3: test the latest delete mark before the safe point should be cleaned.
func (c *client) testGcLatestStaleDeleteMark(shouldGc bool) {
	var err error
	debugCli := c.mustGetTiKVCtlClient(c.kvAddrs[0])

	key := append([]byte("test_gc_latest_stale_delete"), genRandomBytes(8)...)
	value := []byte("test_gc_latest_stale_delete")

	safepoint := c.mustGetSafePoint()
	physical := safepoint >> 18
	oldStartTs := (physical - timeFence) << 18
	oldCommitTs := oldStartTs + 1000
	newStartTs := (physical - timeFence + 10) << 18
	newCommitTs := newStartTs + 1000

	oldKey, oldValue := genMvccPut(key, value, oldStartTs, oldCommitTs)
	err = c.rawKvCli.Put(context.TODO(), oldKey, oldValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	newKey, newValue := genMvccDelete(key, newStartTs, newCommitTs)
	err = c.rawKvCli.Put(context.TODO(), newKey, newValue, rawkv.PutOption{Cf: rawkv.CfWrite})
	if err != nil {
		log.Fatalf("[%s] kv put error %v", caseLabel, err)
	}

	gcByCompact(debugCli)

	type Pair struct {
		key   []byte
		value []byte
	}

	for _, pair := range []Pair{Pair{oldKey, oldValue}, Pair{newKey, newValue}} {
		value, err = c.rawKvCli.Get(context.TODO(), pair.key, rawkv.GetOption{Cf: rawkv.CfWrite})
		if err != nil {
			log.Fatalf("[%s] kv get error %v", caseLabel, err)
		}
		if shouldGc && value != nil {
			log.Fatalf("[%s] the version should be cleaned", caseLabel)
		} else if !shouldGc && !sliceEqual(pair.value, value) {
			log.Fatalf("[%s] the version shouldn't be cleaned", caseLabel)
		}
	}
}

func (c *client) testDynamicConfChange() {
	var err error

	debugCli := c.mustGetTiKVCtlClient(c.kvAddrs[0])
	gcByCompact(debugCli) // Regenerate all sst files.

	log.Infof("[%s] Testing change gc.ratio-threshold to 0.9", caseLabel)
	_, err = c.db.Exec("set config tikv gc.ratio-threshold = 0.9")
	if err != nil {
		log.Fatalf("[%s] change gc.ratio-threshold fail, error %v", caseLabel, err)
	}
	time.Sleep(5 * time.Second) // Sleep a while to wait TiKVs get the update.
	c.testGcLatestStaleDeleteMark(true)

	if *clusterVersion == "4.x" {
		// Change skip-version-check.
		log.Infof("[%s] Testing change gc.compaction-filter-skip-version-check to false", caseLabel)
		_, err = c.db.Exec("set config tikv gc.`compaction-filter-skip-version-check` = false")
		if err != nil {
			log.Fatalf("[%s] change gc.compaction-filter-skip-version-check fail, error %v", caseLabel, err)
		}
		time.Sleep(5 * time.Second) // Sleep a while to wait TiKVs get the update.
		c.testGcLatestStaleDeleteMark(false)

		log.Infof("[%s] Testing change gc.compaction-filter-skip-version-check to true", caseLabel)
		_, err = c.db.Exec("set config tikv gc.`compaction-filter-skip-version-check` = true")
		if err != nil {
			log.Fatalf("[%s] change gc.compaction-filter-skip-version-check fail, error %v", caseLabel, err)
		}
		time.Sleep(5 * time.Second) // Sleep a while to wait TiKVs get the update.
		c.testGcLatestStaleDeleteMark(true)
	}

	// Change enable-compaction-filter
	log.Infof("[%s] Testing change gc.enable-compaction-filter", caseLabel)
	_, err = c.db.Exec("set config tikv gc.enable-compaction-filter = false")
	if err != nil {
		log.Fatalf("[%s] change gc.enable-compaction-filter fail, error %v", caseLabel, err)
	}
	time.Sleep(5 * time.Second) // Sleep a while to wait TiKVs get the update.
	c.testGcLatestStaleDeleteMark(false)

	log.Infof("[%s] Testing change gc.enable-compaction-filter", caseLabel)
	_, err = c.db.Exec("set config tikv gc.enable-compaction-filter = true")
	if err != nil {
		log.Fatalf("[%s] change gc.enable-compaction-filter fail, error %v", caseLabel, err)
	}
	time.Sleep(5 * time.Second) // Sleep a while to wait TiKVs get the update.
	c.testGcLatestStaleDeleteMark(true)
}

func sliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
