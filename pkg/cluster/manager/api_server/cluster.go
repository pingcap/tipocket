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
	name := vars["name"]
	rr, err := m.Resource.FindResourceRequestItemsByResourceRequestName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request item by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rr)
}

func (m *Manager) clusterRun(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, fmt.Sprintf("read workload result failed: %v", err), http.StatusInternalServerError)
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
		if rr.Status != types.ResourceRequestStatusReady {
			return fmt.Errorf("resource request %s isn't meet", name)
		}
		cr, err := m.Cluster.GetLastClusterRequestByRRID(rr.ID)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return errors.Trace(err)
		}
		if cr != nil && cr.Status != types.ClusterRequestStatusDone {
			return fmt.Errorf("resource request %s is using", name)
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
	cr, err := m.Cluster.GetLastClusterRequestByRRID(rr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	cr.Status = types.ClusterRequestStatusRunning
	if err := m.Cluster.UpdateClusterRequest(m.DB.DB, cr); err != nil {
		fail(w, err)
		return
	}
	rris, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	var rids []uint
	for _, rri := range rris {
		rids = append(rids, rri.RID)
	}
	rs, err := m.Resource.FindResources(m.Resource.DB.DB, "id IN (?)", rids)
	if err != nil {
		fail(w, err)
		return
	}
	crts, err := m.Cluster.FindClusterRequestToposByCRID(cr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	wr, err := m.Cluster.GetClusterWorkloadByClusterRequestID(cr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	wr.Status = types.WorkloadStatusRunning
	if err := m.Cluster.UpdateWorkloadRequest(m.DB.DB, wr); err != nil {
		fail(w, err)
		return
	}
	if err := m.runWorkloadWithBaseline(name, rs, rr, rris, cr, crts, wr); err != nil {
		fail(w, err)
		return
	}
	wr.Status = types.WorkloadStatusDone
	if err := m.Cluster.UpdateWorkloadRequest(m.DB.DB, wr); err != nil {
		fail(w, err)
		return
	}
	var report = "ok"
	wps, err := m.Cluster.FindWorkloadReportsByClusterRequestID(cr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	if len(wps) == 0 {
		ok(w, fmt.Sprintf("success, but no workload report of cr_id %d", cr.ID))
		return
	}
	report = *wps[len(wps)-1].PlainText
	cr.Status = types.ClusterRequestStatusDone
	if err := m.Cluster.UpdateClusterRequest(m.DB.DB, cr); err != nil {
		fail(w, err)
		return
	}
	ok(w, report)
}

func (m *Manager) clusterScaleOut(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	component := vars["component"]

	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	rri, err := m.Resource.GetResourceRequestItemByID(uint(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request item by id %d failed, err: %v", id, err.Error()), http.StatusInternalServerError)
		return
	}
	resource, err := m.Resource.GetResourceByID(rri.RID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource by id %d failed, err: %v", rri.RID, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := deploy.TryScaleOut(name, resource, component); err != nil {
		http.Error(w, fmt.Sprintf("try scale out cluster %s failed, err: %v", name, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := m.setScaleOut(rri, component); err != nil {
		http.Error(w, fmt.Sprintf("scale out failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "scale out cluster %s success", rr.Name)
}

func (m *Manager) clusterScaleIn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	component := vars["component"]

	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	rri, err := m.Resource.GetResourceRequestItemByID(uint(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request item by id %d failed, err: %v", id, err.Error()), http.StatusInternalServerError)
		return
	}
	resource, err := m.Resource.GetResourceByID(rri.RID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource by id %d failed, err: %v", rri.RID, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := deploy.TryScaleIn(name, resource, component); err != nil {
		http.Error(w, fmt.Sprintf("try scale in cluster %s failed, err: %v", name, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := m.setScaleIn(rri, component); err != nil {
		http.Error(w, fmt.Sprintf("scale in failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "scale in cluster %s success", rr.Name)
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
