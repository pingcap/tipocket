package crossregion

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pingcap/log"
	"go.uber.org/zap"

	util2 "github.com/pingcap/tipocket/testcase/cross-region/pkg/util"
)

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
	for i := 1; i <= 3; i++ {
		if err := c.transferPDAllocator(fmt.Sprintf("dc-%d", i)); err != nil {
			return err
		}
	}
	log.Info("new allocators ready")
	if err := c.requestGlobalTSOErr(ctx); err != nil {
		return err
	}
	return c.requestTSOs(ctx)
}

func (c *crossRegionClient) requestTSOs(ctx context.Context) error {
	tsoCtx, tsoCancel := context.WithCancel(ctx)
	defer tsoCancel()
	tsoWg := &sync.WaitGroup{}
	tsoWg.Add(4)
	tsoErrCh := make(chan error, 4)
	go c.requestTSO(tsoCtx, "global", tsoWg, tsoErrCh)
	for i := 1; i <= 3; i++ {
		go c.requestTSO(tsoCtx, fmt.Sprintf("dc-%v", i), tsoWg, tsoErrCh)
	}
	tsoWg.Wait()
	if len(tsoErrCh) < 1 {
		log.Info("requestTSOs success")
		return nil
	}
	var tsoErrors []error
	for len(tsoErrCh) > 0 {
		tsoErr := <-tsoErrCh
		tsoErrors = append(tsoErrors, tsoErr)
	}
	return util2.WrapErrors(tsoErrors)
}

func (c *crossRegionClient) requestTSO(ctx context.Context, dcLocation string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	if c.pdClient != nil {
		for i := 0; i < c.TSORequestTimes; i++ {
			physical, logical, err := c.pdClient.GetLocalTS(ctx, dcLocation)
			if err != nil {
				log.Error("requestTSO failed", zap.String("dc-location", dcLocation), zap.Error(err))
				errCh <- err
				return
			}
			log.Info("request TSO", zap.String("dcLocation", dcLocation),
				zap.Int64("physical", physical),
				zap.Int64("logical", logical))
		}
	}
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
		return fmt.Errorf("dclocation %v don't have enough member", dcLocation)
	}
	transferName := ""
	for _, name := range names {
		if name != allocatorName {
			transferName = name
			break
		}
	}
	if transferName == "" {
		return fmt.Errorf("dclocation %v haven't find transfer pd member", dcLocation)
	}
	err = c.pdHTTPClient.TransferAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator committed", zap.String("dclocation", dcLocation),
		zap.String("target-allocator", transferName),
		zap.String("origin-allocator", allocatorName))
	err = c.waitAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator finish", zap.String("dclocation", dcLocation),
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
		for _, dclocation := range dcLocations {
			_, ok := members.TsoAllocatorLeaders[dclocation]
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
	})
}

func (c *crossRegionClient) requestGlobalTSOErr(ctx context.Context) error {
	_, _, err := c.pdClient.GetTS(ctx)
	if err == nil {
		return fmt.Errorf("global tso should return error")
	}
	if !strings.Contains(err.Error(), "mismatch leader id") {
		return fmt.Errorf("global tso should return mismatch leader id error")
	}
	return nil
}
