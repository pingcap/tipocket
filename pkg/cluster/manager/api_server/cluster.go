package api_server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/jpillora/backoff"
	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// PollPendingClusterRequests polls pending cluster requests, bind them to resource requests which are idle.
func (m *Manager) PollPendingClusterRequests(ctx context.Context) {
	b := &backoff.Backoff{
		Min:    1 * time.Second,
		Max:    10 * time.Second,
		Factor: 1.2,
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
			crs, err := m.Cluster.FindClusterRequests(tx, "status = ?", types.ClusterRequestStatusPending)
			if err != nil {
				return err
			}
			for _, cr := range crs {
				rr, err := m.Resource.FindResourceRequest(tx, cr.RRID)
				if err != nil {
					return err
				}
				if rr.Status != types.ResourceRequestStatusIdle {
					continue
				}
				rr.CRID = cr.ID
				rr.Status = types.ResourceRequestStatusPending
				// don't update cluster request here because this cluster request is still in pending until the
				// resource requests getting enough resources
				if err := m.Resource.UpdateResourceRequest(tx, rr); err != nil {
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

// PollReadyClusterRequests polls ready cluster request and schedules workload of it
func (m *Manager) PollReadyClusterRequests(ctx context.Context) {
	b := &backoff.Backoff{
		Min:    1 * time.Second,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}
	for {
	ENTRY:
		time.Sleep(b.Duration())
		select {
		case <-ctx.Done():
			return
		default:
		}
		crs, err := m.Cluster.FindClusterRequests(m.Cluster.DB.DB, "status = ?", types.ClusterRequestStatusReady)
		if err != nil {
			zap.L().Error("find cluster requests failed", zap.Error(errors.Trace(err)))
			continue
		}
		if len(crs) == 0 {
			continue
		}
		for _, cr := range crs {
			if err := m.scheduleClusterWorkload(cr); err != nil {
				zap.L().Error("schedule cluster workload failed", zap.Uint("cr_id", cr.ID), zap.Error(err))
				goto ENTRY
			}
		}
		b.Reset()
	}
}

func (m *Manager) scheduleClusterWorkload(cr *types.ClusterRequest) error {
	err := m.Cluster.DB.Transaction(func(tx *gorm.DB) error {
		cr, err := m.Cluster.GetClusterRequest(tx, cr.ID)
		if err != nil {
			return err
		}
		if cr.Status != types.ClusterRequestStatusReady {
			return fmt.Errorf("expect cluster request in `READY` state, but got %s", cr.Status)
		}
		cr.Status = types.ClusterRequestStatusRunning
		return m.Cluster.UpdateClusterRequest(tx, cr)
	})
	if err != nil {
		return err
	}
	go func() {
		err := m.runWorkload(cr)
		if err != nil {
			zap.L().Error("run workload failed", zap.Uint("cr_id", cr.ID), zap.Error(err))
		}
	}()
	return nil
}

func (m *Manager) clusterList(w http.ResponseWriter, r *http.Request) {
	cluster, err := m.Cluster.FindClusterRequests(m.Cluster.DB.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cluster)
}

func (m *Manager) clusterResourceByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID, err := strconv.ParseUint(vars["cluster_id"], 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	rr, err := m.Resource.FindResourceRequestItemsByClusterRequestID(uint(clusterID))
	if err != nil {
		fail(w, err)
		return
	}
	json.NewEncoder(w).Encode(rr)
}

func (m *Manager) submitClusterRequest(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		ClusterRequest      types.ClusterRequest            `json:"cluster_request"`
		ClusterRequestTopos []*types.ClusterRequestTopology `json:"cluster_request_topologies"`
		Workload            *types.WorkloadRequest          `json:"cluster_workload"`
	}
	vars := mux.Vars(r)
	name := vars["name"]
	var request Request

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1000000))
	if err != nil {
		fail(w, fmt.Errorf("read workload result failed: %v", err))
		return
	}
	r.Body.Close()
	if err := json.Unmarshal(body, &request); err != nil {
		fail(w, err)
		return
	}
	var rr *types.ResourceRequest
	err = m.DB.Transaction(func(tx *gorm.DB) error {
		rr, err = m.Resource.GetResourceRequestByName(tx, name)
		if err != nil {
			return errors.Trace(err)
		}
		request.ClusterRequest.RRID = rr.ID
		if err := m.Cluster.CreateClusterRequest(tx, &request.ClusterRequest); err != nil {
			return errors.Trace(err)
		}
		for _, crt := range request.ClusterRequestTopos {
			crt.CRID = request.ClusterRequest.ID
		}
		if err := m.Cluster.CreateClusterRequestTopos(tx, request.ClusterRequestTopos); err != nil {
			return errors.Trace(err)
		}
		request.Workload.CRID = request.ClusterRequest.ID
		if err := m.Cluster.CreateWorkloadRequest(tx, request.Workload); err != nil {
			return errors.Trace(err)
		}
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	okJSON(w, map[string]interface{}{
		"cluster_request_id": request.ClusterRequest.ID,
	})
}

func (m *Manager) queryClusterRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID, err := strconv.ParseUint(vars["cluster_id"], 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	cr, err := m.Cluster.GetClusterRequest(m.Cluster.DB.DB, uint(clusterID))
	if err != nil {
		fail(w, err)
		return
	}
	okJSON(w, cr)
}

func (m *Manager) clusterScaleOut(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterRequestID, _ := strconv.ParseUint(vars["cluster_id"], 10, 64)
	id, _ := strconv.ParseUint(vars["id"], 10, 64)
	component := vars["component"]
	cr, err := m.Cluster.GetClusterRequest(m.Cluster.DB.DB, uint(clusterRequestID))
	if err != nil {
		fail(w, err)
		return
	}
	rri, err := m.Resource.GetResourceRequestItemByID(uint(id))
	if err != nil {
		fail(w, err)
		return
	}
	resource, err := m.Resource.GetResourceByID(rri.RID)
	if err != nil {
		fail(w, err)
		return
	}
	if err := deploy.TryScaleOut(cr.Name, resource, component); err != nil {
		fail(w, err)
		return
	}
	if err := m.setScaleOut(rri, component); err != nil {
		fail(w, err)
		return
	}
	ok(w, fmt.Sprintf("scale out cluster %d success", clusterRequestID))
}

func (m *Manager) clusterScaleIn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterRequestID, _ := strconv.ParseUint(vars["cluster_id"], 10, 64)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	component := vars["component"]
	cr, err := m.Cluster.GetClusterRequest(m.Cluster.DB.DB, uint(clusterRequestID))
	if err != nil {
		fail(w, err)
		return
	}
	rri, err := m.Resource.GetResourceRequestItemByID(uint(id))
	if err != nil {
		fail(w, err)
		return
	}
	resource, err := m.Resource.GetResourceByID(rri.RID)
	if err != nil {
		fail(w, err)
		return
	}
	if err := deploy.TryScaleIn(cr.Name, resource, component); err != nil {
		fail(w, err)
		return
	}
	if err := m.setScaleIn(rri, component); err != nil {
		fail(w, err)
		return
	}
	ok(w, fmt.Sprintf("scale in cluster %d success", clusterRequestID))
}

func (m *Manager) setOnline(rris []*types.ResourceRequestItem, crts []*types.ClusterRequestTopology) error {
	rriItem2RRi := make(map[uint]*types.ResourceRequestItem)
	for idx, rri := range rris {
		rriItem2RRi[rri.ItemID] = rris[idx]
		rri.Components = ""
	}
	for _, crt := range crts {
		crt.Status = types.ClusterTopoStatusOnline
		rri := rriItem2RRi[crt.RRIItemID]
		if len(rri.Components) == 0 {
			rri.Components = crt.Component
		} else {
			rri.Components = strings.Join([]string{rri.Components, crt.Component}, "|")
		}
	}
	return m.Resource.UpdateResourceRequestItemsAndClusterRequestTopos(rris, crts)
}

func (m *Manager) setScaleOut(rri *types.ResourceRequestItem, component string) error {
	if len(rri.Components) == 0 {
		rri.Components = component
	} else {
		rri.Components = strings.Join([]string{rri.Components, component}, "|")
	}
	return m.Resource.UpdateResourceRequestItemsAndClusterRequestTopos([]*types.ResourceRequestItem{rri}, nil)
}

func (m *Manager) setScaleIn(rri *types.ResourceRequestItem, component string) error {
	var components []string
	for _, c := range strings.Split(rri.Components, "|") {
		if c != component {
			components = append(components, c)
		}
	}
	rri.Components = strings.Join(components, "|")
	return m.Resource.UpdateResourceRequestItemsAndClusterRequestTopos([]*types.ResourceRequestItem{rri}, nil)
}

func (m *Manager) setOffline(rris []*types.ResourceRequestItem, crts []*types.ClusterRequestTopology) error {
	for _, rri := range rris {
		rri.Components = ""
	}
	for _, crt := range crts {
		crt.Status = types.ClusterTopoStatusReady
	}
	return m.Resource.UpdateResourceRequestItemsAndClusterRequestTopos(rris, crts)
}
