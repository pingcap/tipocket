package api_server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/workload"
)

func (m *Manager) runWorkload(cr *types.ClusterRequest) error {
	rr, err := m.Resource.GetResourceRequest(m.Resource.DB.DB, cr.RRID)
	if err != nil {
		return errors.Trace(err)
	}
	rs, err := m.Resource.FindResources(m.Resource.DB.DB, "rr_id = ?", rr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	rris, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	crts, err := m.Cluster.FindClusterRequestToposByCRID(cr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	wr, err := m.Cluster.GetClusterWorkloadByClusterRequestID(cr.ID)
	if err != nil {
		return errors.Trace(err)
	}
	wr.Status = types.WorkloadStatusRunning
	if err := m.Cluster.UpdateWorkloadRequest(m.DB.DB, wr); err != nil {
		return errors.Trace(err)
	}
	// FIXME(@mahjonp): add type field on workloads
	if m.runWorkloadWithBaseline(rs, rr, rris, cr, crts, wr); err != nil {
		return errors.Trace(err)
	}
	wr.Status = types.WorkloadStatusDone
	if err := m.Cluster.UpdateWorkloadRequest(m.DB.DB, wr); err != nil {
		return errors.Trace(err)
	}
	// mark cluster request finished
	cr.Status = types.ClusterRequestStatusDone
	if err := m.Cluster.UpdateClusterRequest(m.Cluster.DB.DB, cr); err != nil {
		goto ERR
	}
	rr.Status = types.ResourceRequestStatusIdle
	rr.CRID = 0
	if err := m.Resource.UpdateResourceRequest(m.Resource.DB.DB, rr); err != nil {
		goto ERR
	}
	for _, r := range rs {
		r.Status = types.ResourceStatusReady
		r.RRID = 0
		if err := m.Resource.UpdateResource(m.Resource.DB.DB, r); err != nil {
			goto ERR
		}
	}
	for _, rri := range rris {
		rri.Components = ""
		rri.RID = 0
	}
	if err := m.Resource.UpdateResourceRequestItems(m.Resource.DB.DB, rris); err != nil {
		return errors.Trace(err)
	}
	return nil
ERR:
	zap.L().Error("teardown cluster request failed", zap.Uint("cr_id", cr.ID), zap.Error(err))
	return err
}

func (m *Manager) runWorkloadWithBaseline(
	resources []*types.Resource,
	rr *types.ResourceRequest,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {

	err := m.runClusterWorkload(resources, rr, rris, cr, crts, wr)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(m.runClusterWorkload(resources, rr, rris, cr.Baseline(), crts, wr))
}

func (m *Manager) runClusterWorkload(
	resources []*types.Resource,
	rr *types.ResourceRequest,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {
	var (
		err  error
		topo *deploy.Topology
	)
	if topo, err = deploy.TryDeployCluster(rr.Name, resources, rris, cr, crts); err != nil {
		return errors.Trace(err)
	}
	if err := m.setOnline(rris, crts); err != nil {
		return errors.Trace(err)
	}
	zap.L().Info("deploy and start cluster success",
		zap.Uint("cr_id", cr.ID))

	_, _, err = workload.TryRunWorkload(rr.Name, resources, rris, wr, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if err = m.archiveArtifacts(rr.ID, topo); err != nil {
		return errors.Trace(err)
	}
	if err = deploy.TryDestroyCluster(rr.Name); err != nil {
		return errors.Trace(err)
	}
	if err = m.setOffline(rris, crts); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (m *Manager) uploadWorkloadResult(w http.ResponseWriter, r *http.Request) {
	type Result struct {
		Data      string  `json:"data"`
		PlainText *string `json:"plaintext"`
	}
	vars := mux.Vars(r)
	name := vars["name"]

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1000000))
	if err != nil {
		http.Error(w, fmt.Sprintf("read workload result failed: %v", err), http.StatusInternalServerError)
	}
	r.Body.Close()
	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, fmt.Sprintf("unmarshal workload result failed: %v", err), http.StatusInternalServerError)
	}
	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetLastClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by resource request id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	//if cr.Status != types.ClusterRequestStatusReady {
	//	http.Error(w, fmt.Sprintf("cluster %s expect %s, but got %s", rr.Name, types.ClusterRequestStatusReady, cr.Status), http.StatusInternalServerError)
	//	return
	//}
	wr := &types.WorkloadReport{
		CRID:      cr.ID,
		Data:      result.Data,
		PlainText: result.PlainText,
	}
	if err := m.Cluster.CreateWorkloadReport(wr); err != nil {
		http.Error(w, fmt.Sprintf("upload workload result %+v failed: %v", wr, err), http.StatusInternalServerError)
		return
	}
	ok(w, "upload workload result success")
}

func (m *Manager) getWorkloadResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetLastClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by rr_id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	result, err := m.Cluster.FindWorkloadReportsByClusterRequestID(cr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	okJSON(w, result)
}
