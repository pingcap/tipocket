package titan

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	pd "github.com/pingcap/pd/client"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// CaseCreator is a creator of a read-stress test
type CaseCreator struct {
}

// Create creates a read-stress test client
func (c CaseCreator) Create(node types.ClientNode) core.Client {
	return &titanClient{}
}

type titanClient struct {
	cli *rawkv.Client
	pd  pd.Client
}

// SetUp implements the core.Client interface.
func (c *titanClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	log.Infof("setup client %v start", idx)

	clusterName := nodes[0].ClusterName
	ns := nodes[0].Namespace
	pdAddrs := []string{fmt.Sprintf("%s-pd.%s.svc:2379", clusterName, ns)}
	if len(pdAddrs) == 0 {
		return errors.New("No pd node found")
	}

	cfg := config.Default()
	cfg.Raw.MaxScanLimit = 40000
	var err error
	c.cli, err = rawkv.NewClient(ctx, pdAddrs, cfg)
	if err != nil {
		log.Fatalf("create tikv client error: %v", err)
	}

	log.Infof("setup client %v end", idx)

	return nil
}

// TearDown implements the core.Client interface.
func (c *titanClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

func (c *titanClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

func (c *titanClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *titanClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *titanClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
	for {
		start := []byte(fmt.Sprintf("v%03d", 1))
		end := []byte(fmt.Sprintf("v%03d", 101))
		err := c.cli.DeleteRange(ctx, start, end)
		if err != nil {
			log.Fatalf("delete range %v - %v failed: %v", start, end, err)
		}

		for i := 1; i <= 100; i++ {
			rnd := rand.New(rand.NewSource(time.Now().Unix()))
			keyHashMap := make(map[string]uint32)
			hashs := make(map[uint32]struct{})

			for j := 0; j <= 40000; j++ {
				key := fmt.Sprintf("v%03dj%05d", i, j)
				vallen := 0
				if j%2 == 0 {
					vallen = 30 * 1024
				} else {
					vallen = 20 * 1024
				}
				val := make([]byte, vallen)
				for {
					util.RandString(val, rnd)
					h32 := util.Hashfnv32a(val)
					if _, ok := hashs[h32]; !ok {
						keyHashMap[key] = h32
						hashs[h32] = struct{}{}
						break
					}
				}
				err := c.cli.Put(ctx, []byte(key), val)
				if err != nil {
					log.Fatalf("put key %v value %v failed: %v", key, val, err)
				}

				if j != 0 && j%1000 == 0 {
					for k := j - 1000; k < j; k += 5 {
						err = c.cli.Delete(ctx, []byte(fmt.Sprintf("v%03dj%05d", i, k)))
						if err != nil {
							log.Fatalf("delete key %v failed: %v", key, err)
						}
					}
				}
			}

			var keys [][]byte
			var values [][]byte
			var err error
			start := []byte(fmt.Sprintf("v%03dj%05d", i, 0))
			end := []byte(fmt.Sprintf("v%03dj%05d", i, 40000))
			if rnd.Int31()%2 == 0 {
				keys, values, err = c.cli.Scan(ctx, start, end, 40000)
			} else {
				keys, values, err = c.cli.ReverseScan(ctx, start, end, 40000)
			}
			if err != nil {
				log.Fatalf("scan keys %v - %v failed: %v", start, end, err)
			}
			if len(keys) != 40000-40000/5 {
				log.Fatalf("wrong count of keys, want: %v, get: %v", 40000-40000/5, len(keys))
			}
			for l, key := range keys {
				if hash, ok := keyHashMap[string(key)]; ok {
					h32 := util.Hashfnv32a(values[l])
					if h32 != hash {
						log.Fatalf("unexpected value %v of key %v, want hash: %v, get hash:%v", values[l], key, hash, h32)
					}
				} else {
					log.Fatalf("unexpected key %v", key)
				}
			}

			select {
			case <-ctx.Done():
				log.Info("Test finished.")
				return nil
			default:
			}

			start = []byte(fmt.Sprintf("v%03d", i))
			end = []byte(fmt.Sprintf("v%03d", i+1))
			err = c.cli.DeleteRange(ctx, start, end)
			if err != nil {
				log.Fatalf("delete range %v - %v failed: %v", start, end, err)
			}
		}
	}
}
