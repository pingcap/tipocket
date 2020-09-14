package api_server

import (
	"context"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jpillora/backoff"
	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// PollPendingResourceRequests polls pending resource requests and match them with all idle resources.
// When manager finds that one resource request have enough idle matching resources, it binds these resources to this resource request,
// change this resource request to `READY` state so that resource request can be scheduled to the cluster request that have bound to it,
// and then mark this cluster request to `READY` state.
func (m *Manager) PollPendingResourceRequests(ctx context.Context) {
	b := &backoff.Backoff{
		Min:    1 * time.Second,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}
	for {
		time.Sleep(b.Duration())
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := m.Resource.DB.Transaction(func(tx *gorm.DB) error {
			rrs, err := m.Resource.FindResourceRequests(tx, "status = ?", types.ClusterRequestStatusPending)
			if err != nil {
				return err
			}
			for _, rr := range rrs {
				if err := m.tryAllocateResourcesToRequest(tx, rr); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			zap.L().Error("find cluster requests failed", zap.Error(err))
			continue
		}
		b.Reset()
	}

}

// tryAllocateResourcesToRequest
func (m *Manager) tryAllocateResourcesToRequest(tx *gorm.DB, rr *types.ResourceRequest) error {
	items, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	instanceTypeToResources := make(map[string][]*types.Resource)
	rs, err := m.Resource.FindResources(tx, "status = ?", types.ResourceStatusReady)
	if err != nil {
		return errors.Trace(err)
	}
	for _, r := range rs {
		if _, ok := instanceTypeToResources[r.InstanceType]; !ok {
			instanceTypeToResources[r.InstanceType] = make([]*types.Resource, 0)
		}
		instanceTypeToResources[r.InstanceType] = append(instanceTypeToResources[r.InstanceType], r)
	}
	var bindResources []*types.Resource
	for _, item := range items {
		if len(instanceTypeToResources[item.InstanceType]) == 0 {
			return nil
		}
		r := instanceTypeToResources[item.InstanceType][0]
		instanceTypeToResources[item.InstanceType] = instanceTypeToResources[item.InstanceType][1:]
		bindResources = append(bindResources, r)
		r.RRID = rr.ID
		item.RID = r.ID
	}
	for _, r := range bindResources {
		if err := m.Resource.UpdateResource(tx, r); err != nil {
			return errors.Trace(err)
		}
	}
	if err := m.Resource.UpdateResourceRequestItems(tx, items); err != nil {
		return errors.Trace(err)
	}
	rr.Status = types.ResourceRequestStatusReady
	if err := m.Resource.UpdateResourceRequest(tx, rr); err != nil {
		return errors.Trace(err)
	}
	cr, err := m.Cluster.GetClusterRequest(tx, rr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	if cr.Status != types.ClusterRequestStatusPending {
		return errors.Trace(fmt.Errorf("expect cluster request %d 's state to be %s, but got %s", cr.ID,
			types.ClusterRequestStatusPending, cr.Status))
	}
	cr.Status = types.ClusterRequestStatusReady
	return errors.Trace(m.Cluster.UpdateClusterRequest(tx, cr))
}
