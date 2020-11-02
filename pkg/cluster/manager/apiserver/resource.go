package apiserver

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
		Factor: 1.3,
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
				return errors.Trace(err)
			}
			rrqs, err := m.Resource.FindResourceRequestQueue(tx)
			if err != nil {
				return errors.Trace(err)
			}
			for _, rr := range rrs {
				if err := m.tryAllocateResourcesToRequest(tx, rr, rrqs); err != nil {
					return errors.Trace(err)
				}
			}
			return nil
		})
		if err != nil {
			zap.L().Error("find pending resource requests failed", zap.Error(err))
			continue
		}
		b.Reset()
	}

}

// tryAllocateResourcesToRequest find suitable resource for each item
// 1. if the item has specified a resource id, the allocator pushes it to the end of that resource's pending list
//    which avoids this resource being binding to other cluster requests(to avoid starvation)
// 2. otherwise, the allocator tries to find a idle resource according to instance type field
func (m *Manager) tryAllocateResourcesToRequest(tx *gorm.DB, rr *types.ResourceRequest, rrqs []*types.ResourceRequestQueue) error {
	id2ResourceMaps := make(map[uint]*types.Resource)
	instanceTypeToResources := make(map[string][]*types.Resource)
	resourceID2ResourceRequestQueue := make(map[uint]*types.ResourceRequestQueue)

	items, err := m.Resource.FindResourceRequestItemsByRRID(tx, rr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	rs, err := m.Resource.FindResources(tx, "status = ? OR status = ?", types.ResourceStatusReady, types.ResourceStatusBinding)
	if err != nil {
		return errors.Trace(err)
	}
	for idx, rrq := range rrqs {
		resourceID2ResourceRequestQueue[rrq.ResourceID] = rrqs[idx]
	}
	for _, r := range rs {
		if _, ok := resourceID2ResourceRequestQueue[r.ID]; !ok {
			resourceID2ResourceRequestQueue[r.ID] = &types.ResourceRequestQueue{ResourceID: r.ID}
		}
	}
	for idx, r := range rs {
		id2ResourceMaps[r.ID] = rs[idx]
		if _, ok := instanceTypeToResources[r.InstanceType]; !ok {
			instanceTypeToResources[r.InstanceType] = make([]*types.Resource, 0)
		}
		instanceTypeToResources[r.InstanceType] = append(instanceTypeToResources[r.InstanceType], r)
	}

	var (
		bindResources    []*types.Resource
		pendingResources []*types.Resource
	)
BindItem:
	for _, item := range items {
		var r *types.Resource
		// if the request item has specified a resource id
		if item.Static {
			r = id2ResourceMaps[item.RID]
			if r == nil {
				return fmt.Errorf("no online resource %d found", item.RID)
			}
			pendingList := resourceID2ResourceRequestQueue[r.ID].PendingRequestIDList()
			if !pendingList.Exist(item.RRID) {
				resourceID2ResourceRequestQueue[r.ID].PushPendingRequestID(item.RRID)
				pendingList = resourceID2ResourceRequestQueue[r.ID].PendingRequestIDList()
				pendingResources = append(pendingResources, r)
			}
			if !pendingList.IsFirst(item.RRID) {
				continue
			}
			if r.Status != types.ResourceStatusReady {
				continue
			}
		} else {
			for {
				if len(instanceTypeToResources[item.InstanceType]) == 0 {
					continue BindItem
				}
				r = instanceTypeToResources[item.InstanceType][0]
				instanceTypeToResources[item.InstanceType] = instanceTypeToResources[item.InstanceType][1:]
				if !resourceID2ResourceRequestQueue[r.ID].PendingRequestIDList().IsEmpty() {
					continue
				}
				if r.Status != types.ResourceStatusReady {
					continue
				}
				item.RID = r.ID
				break
			}
		}
		bindResources = append(bindResources, r)
		r.RRID = rr.ID
		r.Status = types.ResourceStatusBinding
	}
	// 1. if not all resources are met, just update resource queue those has been specified
	if len(bindResources) != len(items) {
		for _, r := range pendingResources {
			if err := m.Resource.UpdateResourceRequestQueue(tx, resourceID2ResourceRequestQueue[r.ID]); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}
	// 2. for the case which all request items are met
	for _, r := range bindResources {
		if !resourceID2ResourceRequestQueue[r.ID].PendingRequestIDList().IsEmpty() {
			rrID := resourceID2ResourceRequestQueue[r.ID].PopPendingRequestID()
			if rrID != rr.ID {
				panic(fmt.Sprintf("a fatal bug exists here"))
			}
		}
		if err := m.Resource.UpdateResource(tx, r); err != nil {
			return errors.Trace(err)
		}
		if err := m.Resource.UpdateResourceRequestQueue(tx, resourceID2ResourceRequestQueue[r.ID]); err != nil {
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
	cr, err := m.Cluster.GetClusterRequest(tx, rr.CRID)
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
