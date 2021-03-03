package crossregion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingcap/log"
	"go.uber.org/zap"

	util2 "github.com/pingcap/tipocket/testcase/cross-region/pkg/util"
)

const (
	dcLocationNumber  = 3
	physicalShiftBits = 18
	logicalMask       = (1 << physicalShiftBits) - 1
	suffixMask        = logicalMask >> (physicalShiftBits - 2) // Log2(3+1) = 2
)

// TSO wraps the tso response
type TSO struct {
	physical   int64
	logical    int64
	dcLocation string
}

var (
	recordTSO = make(map[TSO]struct{}, 0)
	lastTSO   = sync.Map{} // stored as [string]*TSO, dcLocation -> *TSO
	tsoSuffix = sync.Map{} // stored as [string]int, dcLocation -> suffix
)

func getLastTSO(dcLocation string) (int64, int64) {
	tso, ok := lastTSO.Load(dcLocation)
	if !ok {
		return 0, 0
	}
	return tso.(*TSO).physical, tso.(*TSO).logical
}

func setLastTSO(dcLocation string, physical, logical int64) {
	lastTSO.Store(dcLocation, &TSO{
		physical:   physical,
		logical:    logical,
		dcLocation: dcLocation,
	})
}

func checkTSOSuffix(dcLocation string, suffix int) (int, bool) {
	suffixStored, loaded := tsoSuffix.LoadOrStore(dcLocation, suffix)
	if !loaded {
		return suffix, true
	}
	oldSuffix := suffixStored.(int)
	return oldSuffix, oldSuffix == suffix
}

func (c *crossRegionClient) testTSO(ctx context.Context) error {
	if err := c.requestTSOs(ctx); err != nil {
		return err
	}
	log.Info("start to transfer Leader")
	if err := c.transferLeader(); err != nil {
		return err
	}
	log.Info("leader transfer committed")
	if err := c.waitLeaderReady(); err != nil {
		return err
	}
	log.Info("new leader ready")
	for i := 1; i <= dcLocationNumber; i++ {
		if err := c.transferPDAllocator(fmt.Sprintf("dc-%d", i)); err != nil {
			return err
		}
	}
	log.Info("new allocators ready")
	// after transfer allocator and leader, we can tolerate tso failed once for each dc.
	c.requestTSOAfterTransfer(ctx, "global")
	c.requestTSOAfterTransfer(ctx, "dc-1")
	c.requestTSOAfterTransfer(ctx, "dc-2")
	c.requestTSOAfterTransfer(ctx, "dc-3")
	return c.requestTSOs(ctx)
}

func (c *crossRegionClient) requestTSOs(ctx context.Context) error {
	tsoCtx, tsoCancel := context.WithCancel(ctx)
	defer tsoCancel()
	tsoWg := &sync.WaitGroup{}
	tsoWg.Add(4)
	tsoErrCh := make(chan error, dcLocationNumber+1)
	tsoCh := make(chan TSO, (dcLocationNumber+1)*c.TSORequests)
	go c.requestTSO(tsoCtx, "global", tsoWg, tsoCh, tsoErrCh)
	for i := 1; i <= dcLocationNumber; i++ {
		go c.requestTSO(tsoCtx, fmt.Sprintf("dc-%v", i), tsoWg, tsoCh, tsoErrCh)
	}
	tsoWg.Wait()
	var tsoErrors []error
	if len(tsoErrCh) < 1 {
		log.Info("requestTSOs success")
	} else {
		log.Warn("requestTSOs meets error")
		for len(tsoErrCh) > 0 {
			tsoErrors = append(tsoErrors, <-tsoErrCh)
		}
	}
	// Check uniqueness
	for len(tsoCh) > 0 {
		tsoRes := <-tsoCh
		key := TSO{physical: tsoRes.physical, logical: tsoRes.logical, dcLocation: tsoRes.dcLocation}
		_, ok := recordTSO[key]
		if ok {
			tsoErrors = append(tsoErrors,
				fmt.Errorf("tso repeated, physcial: %v, logical: %v, dc-location %v",
					tsoRes.physical, tsoRes.logical, tsoRes.dcLocation))
			break
		}
	}
	if len(tsoErrors) > 0 {
		return util2.WrapErrors(tsoErrors)
	}
	return nil
}

func (c *crossRegionClient) requestTSO(ctx context.Context, dcLocation string, wg *sync.WaitGroup, tsoCh chan<- TSO, errCh chan<- error) {
	defer wg.Done()
	if c.pdClient == nil {
		log.Error("pd client is not set up", zap.String("dc-location", dcLocation))
		return
	}
	lastPhysical, lastLogical := getLastTSO(dcLocation)
	for i := 0; i < c.TSORequests; i++ {
		physical, logical, err := c.pdClient.GetLocalTS(ctx, dcLocation)
		if err != nil {
			log.Error("requestTSO failed", zap.String("dc-location", dcLocation), zap.Error(err))
			errCh <- err
			return
		}
		log.Info("request TSO", zap.String("dcLocation", dcLocation),
			zap.Int64("physical", physical),
			zap.Int64("logical", logical))
		// assert tso won't fallback
		if physical < lastPhysical || (physical == lastPhysical && logical <= lastLogical) {
			errCh <- fmt.Errorf("tso fallback phsical: %v, logical: %v, dc-location: %v,last-phsical: %v, last-logical: %v",
				physical, logical, dcLocation, lastPhysical, lastLogical)
			return
		}
		newSuffix := int(logical & suffixMask)
		oldSuffix, ok := checkTSOSuffix(dcLocation, newSuffix)
		if !ok {
			errCh <- fmt.Errorf("tso suffix changed, oldSuffix: %v, newSuffix: %v, dc-location: %v",
				oldSuffix, newSuffix, dcLocation)
			return
		}
		tsoCh <- TSO{physical: physical, logical: logical, dcLocation: dcLocation}
		lastPhysical, lastLogical = physical, logical
	}
	setLastTSO(dcLocation, lastPhysical, lastLogical)
}

func (c *crossRegionClient) transferPDAllocator(dcLocation string) error {
	members, err := c.pdHTTPClient.GetMembers()
	if err != nil {
		return err
	}
	var names []string
	for _, member := range members.Members {
		if member.DcLocation == dcLocation {
			names = append(names, member.Name)
		}
	}
	allocatorName := ""
	for dcLocation, allocator := range members.TsoAllocatorLeaders {
		if dcLocation == dcLocation {
			allocatorName = allocator.Name
			break
		}
	}
	if allocatorName == "" || len(names) < 2 {
		return fmt.Errorf("dc-location %v don't have enough member", dcLocation)
	}
	transferName := ""
	for _, name := range names {
		if name != allocatorName {
			transferName = name
			break
		}
	}
	if transferName == "" {
		return fmt.Errorf("dc-location %v haven't find transfer pd member", dcLocation)
	}
	err = c.pdHTTPClient.TransferAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator committed", zap.String("dc-location", dcLocation),
		zap.String("target-allocator", transferName),
		zap.String("origin-allocator", allocatorName))
	err = c.waitAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator finish", zap.String("dc-location", dcLocation),
		zap.String("target-allocator", transferName))
	return nil
}

func (c *crossRegionClient) transferLeader() error {
	members, err := c.pdHTTPClient.GetMembers()
	if err != nil {
		return err
	}
	targetLeader := ""
	for _, member := range members.Members {
		if member.Name != members.Leader.Name {
			err = c.pdHTTPClient.TransferPDLeader(member.Name)
			if err != nil {
				return err
			}
			targetLeader = member.Name
			break
		}
	}
	return c.waitLeader(targetLeader)
}

func (c *crossRegionClient) waitLeaderReady() error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		return members.Leader != nil
	})
}

func (c *crossRegionClient) waitAllocatorReady(dcLocations []string) error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		for _, dcLocation := range dcLocations {
			_, ok := members.TsoAllocatorLeaders[dcLocation]
			if !ok {
				return false
			}
		}
		return true
	})
}

func (c *crossRegionClient) waitLeader(name string) error {
	return util2.WaitUntil(func() bool {
		mems, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		if mems.Leader == nil || mems.Leader.Name != name {
			return false
		}
		return true
	})
}

func (c *crossRegionClient) waitAllocator(name, dcLocation string) error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		for dc, member := range members.TsoAllocatorLeaders {
			if dcLocation == dc && member.Name == name {
				return true
			}
		}
		return false
	}, util2.WithSleepInterval(5*time.Second), util2.WithRetryTimes(30))
}

func (c *crossRegionClient) requestTSOAfterTransfer(ctx context.Context, dc string) {
	_, _, err := c.pdClient.GetLocalTS(ctx, dc)
	if err != nil {
		log.Info("requestTSOAfterTransfer failed", zap.Error(err), zap.String("dcLocation", dc))
	} else {
		log.Info("requestTSOAfterTransfer success", zap.String("dcLocation", dc))
	}
}
