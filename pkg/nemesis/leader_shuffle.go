package nemesis

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/nemesis/fake_kvproto/metapb"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
)

type leaderShuffleGenerator struct {
	name string
}

// NewLeaderShuffleGenerator ...
func NewLeaderShuffleGenerator(name string) *leaderShuffleGenerator {
	return &leaderShuffleGenerator{name: name}
}

// Generate generates container-kill actions, to simulate the case that node can be recovered quickly after being killed
func (l leaderShuffleGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	duration := 10 * time.Second
	return l.schedule(nodes, duration)
}

func (l leaderShuffleGenerator) schedule(nodes []cluster.Node, duration time.Duration) []*core.NemesisOperation {
	nodes = filterComponent(nodes, cluster.PD)
	var ops []*core.NemesisOperation
	ops = append(ops, &core.NemesisOperation{
		Type:        core.PDLeaderShuffler,
		Node:        &nodes[0],
		InvokeArgs:  []interface{}{},
		RecoverArgs: []interface{}{},
		RunTime:     duration,
	})
	return ops
}

func (l leaderShuffleGenerator) Name() string {
	return l.name
}

// LeaderShuffler creates operators in PD to schedule the region containing the regionKey.
type LeaderShuffler struct {
	*pdutil.Client
	regionKey    string
	shuffleFuncs []func() error
}

// NewLeaderShuffler creates a LeaderShuffler to schedule the region containing the regionKey.
func NewLeaderShuffler(pdAddr string, regionKey string) *LeaderShuffler {
	l := new(LeaderShuffler)
	l.Client = pdutil.NewPDClient(http.DefaultClient, pdAddr)
	l.regionKey = regionKey
	l.shuffleFuncs = []func() error{l.transferLeader, l.transferRegion, l.transferOtherRegionToLeader, l.mergeRegion, l.splitRegion}
	return l
}

// Invoke implements Nemesis.Invoke that invokes the nemesis.
func (l *LeaderShuffler) Invoke(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Infof("apply nemesis %s...", core.PDLeaderShuffler)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*time.Duration(10))
	defer cancel()

	pdAddr := fmt.Sprintf("http://%s:%d", node.IP, node.Port)
	l.Client = pdutil.NewPDClient(http.DefaultClient, pdAddr)
	return l.ShuffleLeader()
}

// Recover implements Nemesis.Recover that recovers the nemesis.
func (l *LeaderShuffler) Recover(ctx context.Context, node *cluster.Node, args ...interface{}) error {
	log.Infof("unapply nemesis %s...", core.PDLeaderShuffler)
	return nil
}

// Name implements Nemesis.Name that returns the unique name for the nemesis.
func (l LeaderShuffler) Name() string {
	return string(core.PDLeaderShuffler)
}

// ShuffleLeader create a operator in PD to schedule the region.
func (l *LeaderShuffler) ShuffleLeader() error {
	shuffleFunc := l.shuffleFuncs[rand.Intn(len(l.shuffleFuncs))]
	return shuffleFunc()
}

func (l *LeaderShuffler) transferLeader() error {
	region, err := l.GetRegionByKey(l.regionKey)
	if err != nil {
		return err
	}
	if region == nil || region.Leader == nil {
		log.Infof("[leader shuffler] Can't find the leader of region(%s)", l.regionKey)
		return nil
	}

	var followerIds []uint64
	for _, peer := range region.Peers {
		if peer.GetStoreId() != region.Leader.GetStoreId() {
			followerIds = append(followerIds, peer.GetStoreId())
		}
	}
	if len(followerIds) == 0 {
		log.Warnf("[leader shuffler] [leader=%d] Region #%d has no follower to transfer leader", region.Leader.GetStoreId(), region.ID)
		return nil
	}
	target := followerIds[rand.Intn(len(followerIds))]

	log.Infof("[leader shuffler] [leader=%d] Transfer leader from %d to %d", region.Leader.GetStoreId(), region.Leader.GetStoreId(), target)
	body := make(map[string]interface{})
	body["name"] = "transfer-leader"
	body["region_id"] = region.ID
	body["to_store_id"] = target
	return l.Operators(body)
}

func (l *LeaderShuffler) transferRegion() error {
	stores, err := l.GetStores()
	if err != nil {
		return err
	}
	var onlineStores []uint64
	for _, store := range stores.Stores {
		if metapb.StoreState(store.GetState()) == metapb.StoreState_Up {
			onlineStores = append(onlineStores, store.GetId())
		}
	}

	region, err := l.GetRegionByKey(l.regionKey)
	if err != nil {
		return err
	}
	if region == nil || region.Leader == nil {
		log.Infof("[leader shuffler] Can't find the leader of region(%s)", l.regionKey)
		return nil
	}
	// Remove current leader's id
	for i, id := range onlineStores {
		if id == region.Leader.GetStoreId() {
			onlineStores = append(onlineStores[:i], onlineStores[i+1:]...)
			break
		}
	}
	if len(onlineStores) < len(region.Peers) {
		log.Warnf("[leader shuffler] Don't have enough stores to transfer region, onlineStores: %v", onlineStores)
		return nil
	}

	var current []uint64
	for _, peer := range region.Peers {
		current = append(current, peer.GetStoreId())
	}
	target := onlineStores[:len(current)]
	roles := make([]string, len(target), len(target))
	for i := range roles {
		roles[i] = "voter"
	}

	log.Infof("[leader shuffler] [leader=%d] Transfer leader region #%d from %v to %v", region.Leader.GetStoreId(), region.ID, current, target)
	body := make(map[string]interface{})
	body["name"] = "transfer-region"
	body["region_id"] = region.ID
	body["to_store_ids"] = target
	body["peer_roles"] = roles
	return l.Operators(body)
}

func (l *LeaderShuffler) transferOtherRegionToLeader() error {
	region, err := l.GetRegionByKey(l.regionKey)
	if err != nil {
		return err
	}
	if region == nil || region.Leader == nil {
		log.Infof("[leader shuffler] Can't find the leader of region(%s)", l.regionKey)
		return nil
	}
	regions, err := l.ListRegions()
	if err != nil {
		return err
	}
	var ri *pdutil.RegionInfo
	for _, r := range regions {
		ok := true
		for _, p := range r.Peers {
			if p != nil && p.StoreId == region.Leader.StoreId {
				ok = false
				break
			}
		}
		if ok {
			ri = r
			break
		}
	}
	var toStores []uint64
	if ri == nil {
		log.Infof("[leader shuffler] [leader=%d] No region can be transfer to leader this time", region.Leader.StoreId)
		stores, err := l.GetStores()
		if err != nil {
			return err
		}
		ri = regions[rand.Intn(len(regions))]
		for _, store := range stores.Stores {
			if len(toStores) >= 1 {
				break
			}
			if metapb.StoreState(store.GetState()) == metapb.StoreState_Up && store.Id != region.Leader.StoreId {
				toStores = append(toStores, store.GetId())
			}
		}
		log.Infof("[leader shuffler] [leader=%d] Transfer region #%d from leader to %v", region.Leader.StoreId, ri.ID, toStores)
	} else {
		for i, p := range ri.Peers {
			if i == 0 {
				toStores = append(toStores, region.Leader.StoreId)
			} else {
				toStores = append(toStores, p.StoreId)
			}
		}
		log.Infof("[leader shuffler] [leader=%d] Transfer other region #%d to the leader", region.Leader.StoreId, ri.ID)
	}

	roles := make([]string, len(toStores), len(toStores))
	for i := range roles {
		roles[i] = "voter"
	}

	return l.Operators(map[string]interface{}{
		"name":         "transfer-region",
		"region_id":    ri.ID,
		"to_store_ids": toStores,
		"peer_roles":   roles,
	})
}

func (l *LeaderShuffler) mergeRegion() error {
	region, err := l.GetRegionByKey(l.regionKey)
	if err != nil {
		return err
	}
	if region.StartKey == region.EndKey {
		return nil
	}
	siblings, err := l.GetSiblingRegions(region.ID)
	if err != nil {
		return err
	}
	var target *pdutil.RegionInfo
	for _, sibling := range siblings {
		if sibling != nil {
			target = sibling
			break
		}
	}
	if target == nil {
		log.Warnf("[leader shuffler] [leader=%d] Region #%d doesn't have siblings", region.Leader.GetStoreId(), region.ID)
		return errors.New("Region doesn't have siblings")
	}
	// Randomize target region
	if rand.Intn(2) == 0 {
		region, target = target, region
	}

	log.Infof("[leader shuffler] [leader=%d] Merge leader region #%d and #%d", region.Leader.GetStoreId(), region.ID, target.ID)
	body := make(map[string]interface{})
	body["name"] = "merge-region"
	body["source_region_id"] = region.ID
	body["target_region_id"] = target.ID
	return l.Operators(body)
}

func (l *LeaderShuffler) splitRegion() error {
	region, err := l.GetRegionByKey(l.regionKey)
	if err != nil {
		return err
	}

	log.Infof("[leader shuffler] [leader=%d] Split leader region #%d", region.Leader.GetStoreId(), region.ID)
	body := make(map[string]interface{})
	body["name"] = "split-region"
	body["region_id"] = region.ID
	body["policy"] = "approximate"
	return l.Operators(body)
}
