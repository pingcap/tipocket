package crossregion

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/log"
	"go.uber.org/zap"

	util2 "github.com/pingcap/tipocket/testcase/cross-region/pkg/util"
)

const (
	globalDCLocation  = "global"
	physicalShiftBits = 18
	logicalMask       = (1 << physicalShiftBits) - 1
	suffixMask        = logicalMask >> (physicalShiftBits - 2) // Log2(3+1) = 2
)

type criticalTSOErrorType int

const (
	// fallbackError indicates the TSO fallback occurs.
	fallbackError criticalTSOErrorType = iota
	// uniquenessError indicates the TSO is not unique.
	uniquenessError
	// suffixChangedError indicates the TSO suffix changed unexpectedly.
	suffixChangedError
)

// criticalTSOError represents the critical error which should never occur.
type criticalTSOError struct {
	errType criticalTSOErrorType
	msg     string
}

func newCriticalTSOError(errType criticalTSOErrorType, msg string) *criticalTSOError {
	return &criticalTSOError{
		errType: errType,
		msg:     msg,
	}
}

func (cte *criticalTSOError) Error() string {
	var errTypeMsg string
	switch cte.errType {
	case fallbackError:
		errTypeMsg = "fallbackError"
	case uniquenessError:
		errTypeMsg = "uniquenessError"
	case suffixChangedError:
		errTypeMsg = "suffixChangedError"
	}
	return fmt.Sprintf("criticalTSOError: %s, %s", errTypeMsg, cte.msg)
}

// TSO wraps the tso response
type TSO struct {
	physical   int64
	logical    int64
	dcLocation string
}

var (
	recordTSO = make(map[TSO]struct{})
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

func (c *crossRegionClient) testTSO(ctx context.Context) error {
	if err := c.requestTSOs(ctx); err != nil {
		return err
	}
	log.Info("start to transfer Leader")
	if err := c.transferLeader(); err != nil {
		return err
	}
	log.Info("leader transfer committed and new leader is ready")
	for i := 0; i < len(DCLocations); i++ {
		if err := c.transferPDAllocator(DCLocations[i]); err != nil {
			return err
		}
	}
	log.Info("new allocators ready")
	// after transfer allocator and leader, we can tolerate tso failed once for each dc.
	c.requestTSOAfterTransfer(ctx, globalDCLocation)
	for i := 0; i < len(DCLocations); i++ {
		c.requestTSOAfterTransfer(ctx, DCLocations[i])
	}
	return c.requestTSOs(ctx)
}

func (c *crossRegionClient) requestTSOs(ctx context.Context) error {
	if c.pdClient == nil {
		return fmt.Errorf("pd client is not set up")
	}
	tsoCtx, tsoCancel := context.WithCancel(ctx)
	defer tsoCancel()

	dcLocationNumber := len(DCLocations)
	tsoErrCh := make(chan error, dcLocationNumber+1)
	tsoCh := make(chan TSO, (dcLocationNumber+1)*c.TSORequests)
	tsoWg := &sync.WaitGroup{}
	tsoWg.Add(dcLocationNumber + 1)
	go c.requestGlobalTSO(tsoCtx, DCLocations, tsoWg, tsoCh, tsoErrCh)
	for i := 0; i < dcLocationNumber; i++ {
		go c.requestLocalTSO(tsoCtx, DCLocations[i], tsoWg, tsoCh, tsoErrCh)
	}
	tsoWg.Wait()

	// Check errors.
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
				newCriticalTSOError(
					uniquenessError,
					fmt.Sprintf("tso repeated, physical: %v, logical: %v, dc-location %v",
						tsoRes.physical, tsoRes.logical, tsoRes.dcLocation)))
			break
		}
		recordTSO[key] = struct{}{}
	}
	// Only return the critical errors.
	criticalTSOErrors := make([]error, 0, len(tsoErrors))
	for _, err := range tsoErrors {
		if _, ok := err.(*criticalTSOError); ok {
			criticalTSOErrors = append(criticalTSOErrors, err)
		}
	}
	if len(criticalTSOErrors) > 0 {
		return util2.WrapErrors(criticalTSOErrors)
	}
	return nil
}

func (c *crossRegionClient) requestGlobalTSO(ctx context.Context, dcLocations []string, wg *sync.WaitGroup, tsoCh chan<- TSO, errCh chan<- error) {
	defer wg.Done()
	var (
		err                                   error
		physical, logical                     int64
		oldSuffix                             int
		lastGlobalPhysical, lastGlobalLogical = getLastTSO(globalDCLocation)
		localPhysicalMap                      = make(map[string]int64, len(dcLocations))
		localLogicalMap                       = make(map[string]int64, len(dcLocations))
	)
	for i := 0; i < c.TSORequests; i++ {
		// Get all pre Local TSOs
		for _, dcLocation := range dcLocations {
			localPhysicalMap[dcLocation], localLogicalMap[dcLocation], err = c.pdClient.GetLocalTS(ctx, dcLocation)
			if err != nil {
				log.Error("requestGlobalTSO failed, get pre local tso error", zap.String("dc-location", dcLocation), zap.Error(err))
				errCh <- err
				return
			}
		}
		// Get a Global TSO after
		physical, logical, err = c.pdClient.GetTS(ctx)
		if err != nil {
			log.Error("requestGlobalTSO failed, get global tso error", zap.Error(err))
			errCh <- err
			return
		}
		log.Info("request Global TSO", zap.Int64("physical", physical), zap.Int64("logical", logical))
		// Assert Global TSO won't fallback
		if tsLessEqual(physical, logical, lastGlobalPhysical, lastGlobalLogical) {
			errCh <- newCriticalTSOError(
				fallbackError,
				fmt.Sprintf("tso fallback, dc-location: %s, physical: %d, logical: %d, last-physical: %d, last-logical: %d",
					globalDCLocation, physical, logical, lastGlobalPhysical, lastGlobalLogical))
			return
		}
		// Assert Global TSO is bigger than all pre Local TSOs
		for _, dcLocation := range dcLocations {
			preLocalPhysical := localPhysicalMap[dcLocation]
			preLocalLogical := localLogicalMap[dcLocation]
			if tsLessEqual(physical, logical, preLocalPhysical, preLocalLogical) {
				errCh <- newCriticalTSOError(
					fallbackError,
					fmt.Sprintf("global tso fallback than pre local tso, dc-location: %s, global-physical: %d, global-logical: %d, pre-local-physical: %d, pre-local-logical: %d",
						dcLocation, physical, logical, preLocalPhysical, preLocalLogical))
				return
			}
		}
		// Get all after Local TSOs
		for _, dcLocation := range dcLocations {
			localPhysicalMap[dcLocation], localLogicalMap[dcLocation], err = c.pdClient.GetLocalTS(ctx, dcLocation)
			if err != nil {
				log.Error("requestGlobalTSO failed, get after local tso error", zap.String("dc-location", dcLocation), zap.Error(err))
				errCh <- err
				return
			}
		}
		// Assert all after Local TSOs are bigger than Global TSO
		for _, dcLocation := range dcLocations {
			afterLocalPhysical := localPhysicalMap[dcLocation]
			afterLocalLogical := localLogicalMap[dcLocation]
			if tsLessEqual(afterLocalPhysical, afterLocalLogical, physical, logical) {
				errCh <- newCriticalTSOError(
					fallbackError,
					fmt.Sprintf("after local tso fallback than global tso, dc-location: %s, after-local-physical: %d, after-local-logical: %d, global-physical: %d, global-logical: %d",
						dcLocation, afterLocalPhysical, afterLocalLogical, physical, logical))
				return
			}
		}
		// Check whether the suffix changed.
		oldSuffix, err = checkTSOSuffix(globalDCLocation, logical)
		if err != nil {
			errCh <- err
			return
		}
		log.Info("matched tso suffix", zap.String("dc-location", globalDCLocation), zap.Int("suffix", oldSuffix))
		// Finfish the request.
		tsoCh <- TSO{physical: physical, logical: logical, dcLocation: globalDCLocation}
		lastGlobalPhysical, lastGlobalLogical = physical, logical
		// We should save it ASAP to make other Local TSO request goroutines can use it to compare.
		setLastTSO(globalDCLocation, lastGlobalPhysical, lastGlobalLogical)
	}
}

func tsLessEqual(physical, logical, lastPhysical, lastLogical int64) bool {
	return physical < lastPhysical || (physical == lastPhysical && logical <= lastLogical)
}

func checkTSOSuffix(dcLocation string, logical int64) (int, error) {
	newSuffix := int(logical & suffixMask)
	suffixStored, loaded := tsoSuffix.LoadOrStore(dcLocation, newSuffix)
	if !loaded {
		return newSuffix, nil
	}
	oldSuffix := suffixStored.(int)
	if oldSuffix == newSuffix {
		return oldSuffix, nil
	}
	return oldSuffix, newCriticalTSOError(suffixChangedError, fmt.Sprintf("tso suffix changed, oldSuffix: %d, newSuffix: %d, dc-location: %s", oldSuffix, newSuffix, dcLocation))
}

func (c *crossRegionClient) requestLocalTSO(ctx context.Context, dcLocation string, wg *sync.WaitGroup, tsoCh chan<- TSO, errCh chan<- error) {
	defer wg.Done()
	var (
		err                       error
		physical, logical         int64
		oldSuffix                 int
		lastPhysical, lastLogical = getLastTSO(dcLocation)
	)
	for i := 0; i < c.TSORequests; i++ {
		// Prepare the last Global TSO for the later Local TSO to compare
		var lastGlobalTSOPhysical, lastGlobalTSOLogical int64
		if dcLocation != globalDCLocation {
			lastGlobalTSOPhysical, lastGlobalTSOLogical = getLastTSO(globalDCLocation)
		}
		physical, logical, err = c.pdClient.GetLocalTS(ctx, dcLocation)
		if err != nil {
			log.Error("requestLocalTSO failed", zap.String("dc-location", dcLocation), zap.Error(err))
			errCh <- err
			return
		}
		log.Info("request Local TSO", zap.String("dcLocation", dcLocation), zap.Int64("physical", physical), zap.Int64("logical", logical))
		// assert tso won't fallback
		if tsLessEqual(physical, logical, lastPhysical, lastLogical) {
			errCh <- fmt.Errorf("tso fallback, dc-location: %s, physical: %d, logical: %d, last-physical: %d, last-logical: %d",
				dcLocation, physical, logical, lastPhysical, lastLogical)
			return
		}
		// assert local tso is bigger than last Global TSO
		if tsLessEqual(physical, logical, lastGlobalTSOPhysical, lastGlobalTSOLogical) {
			errCh <- fmt.Errorf("local tso fallback than global, dc-location: %s, physical: %d, logical: %d, last-global-physical: %d, last-global-logical: %d",
				dcLocation, physical, logical, lastGlobalTSOPhysical, lastGlobalTSOLogical)
			return
		}
		// Check whether the suffix changed.
		oldSuffix, err = checkTSOSuffix(dcLocation, logical)
		if err != nil {
			errCh <- err
			return
		}
		log.Info("matched tso suffix", zap.String("dc-location", dcLocation), zap.Int("suffix", oldSuffix))
		// Finfish the request.
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
	for dc, allocator := range members.TsoAllocatorLeaders {
		if dc == dcLocation {
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
	return util2.WaitUntil("PD leader election", func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		return members != nil && members.Leader != nil
	})
}

func (c *crossRegionClient) waitAllocatorReady(dcLocations []string) error {
	return util2.WaitUntil(fmt.Sprintf("%s allocators election", dcLocations), func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		if members != nil && members.TsoAllocatorLeaders != nil {
			for _, dcLocation := range dcLocations {
				_, ok := members.TsoAllocatorLeaders[dcLocation]
				if !ok {
					return false
				}
			}
		}
		return true
	})
}

func (c *crossRegionClient) waitLeader(name string) error {
	for {
		log.Info("Wait for leader election", zap.String("name", name))
		mems, err := c.pdHTTPClient.GetMembers()
		if err != nil && !strings.Contains(err.Error(), "no leader") {
			return err
		}
		if mems != nil && mems.Leader != nil && mems.Leader.Name == name {
			return nil
		}
		time.Sleep(30 * time.Second)
	}
}

func (c *crossRegionClient) waitAllocator(name, dcLocation string) error {
	for {
		log.Info("Wait for allocator election", zap.String("name", name), zap.String("dc-location", dcLocation))
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil && !strings.Contains(err.Error(), "no leader") {
			return err
		}
		if members != nil && members.TsoAllocatorLeaders != nil {
			for dc, member := range members.TsoAllocatorLeaders {
				if dcLocation == dc && member != nil && member.Name == name {
					return nil
				}
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func (c *crossRegionClient) requestTSOAfterTransfer(ctx context.Context, dc string) {
	_, _, err := c.pdClient.GetLocalTS(ctx, dc)
	if err != nil {
		log.Info("requestTSOAfterTransfer failed", zap.Error(err), zap.String("dcLocation", dc))
	} else {
		log.Info("requestTSOAfterTransfer success", zap.String("dcLocation", dc))
	}
}
