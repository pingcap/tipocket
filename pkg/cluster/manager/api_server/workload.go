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

func (m *Manager) runWorkloadWithBaseline(
	name string,
	resources []*types.Resource,
	rr *types.ResourceRequest,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {

	err := m.runClusterWorkload(name, resources, rr, rris, cr, crts, wr)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(m.runClusterWorkload(name, resources, rr, rris, cr.Baseline(), crts, wr))
}

func (m *Manager) runClusterWorkload(name string,
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
		zap.String("name", name),
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
