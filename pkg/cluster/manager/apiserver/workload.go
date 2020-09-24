package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/juju/errors"
	"github.com/rogpeppe/fastuuid"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
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

	workloadFunc := m.runPRWorkload
	switch wr.Type {
	case types.WorkloadTypePR:
		workloadFunc = m.runPRWorkload
	case types.WorkloadTypeStandard:
		workloadFunc = m.runStandardWorkload
	}
	if err := workloadFunc(rs, rris, cr, crts, wr); err != nil {
		return errors.Trace(err)
	}
	wr.Status = types.WorkloadStatusDone
	if err := m.Cluster.UpdateWorkloadRequest(m.DB.DB, wr); err != nil {
		return errors.Trace(err)
	}
	return m.Resource.DB.Transaction(func(tx *gorm.DB) error {
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
			goto ERR
		}
		return nil
	ERR:
		zap.L().Error("teardown cluster request failed", zap.Uint("cr_id", cr.ID), zap.Error(err))
		return err
	})
}

func (m *Manager) runPRWorkload(
	resources []*types.Resource,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {
	err := m.runClusterWorkload(resources, rris, cr, crts, wr)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(m.runClusterWorkload(resources, rris, cr.Baseline(), crts, wr))
}

func (m *Manager) runStandardWorkload(
	resources []*types.Resource,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {
	return m.runClusterWorkload(resources, rris, cr, crts, wr)
}

func (m *Manager) runClusterWorkload(
	resources []*types.Resource,
	rris []*types.ResourceRequestItem,
	cr *types.ClusterRequest,
	crts []*types.ClusterRequestTopology,
	wr *types.WorkloadRequest) error {
	var (
		err            error
		topo           *deploy.Topology
		containerID    string
		out            *bytes.Buffer
		rs             *types.Resource
		dockerExecutor *util.DockerExecutor
	)
	if topo, err = deploy.TryDeployCluster(cr.Name, resources, rris, cr, crts); err != nil {
		return errors.Trace(err)
	}
	if err := m.setOnline(rris, crts); err != nil {
		return errors.Trace(err)
	}
	zap.L().Info("deploy and start cluster success",
		zap.Uint("cr_id", cr.ID))

	artifactUUID := fastuuid.MustNewGenerator().Hex128()
	if wr.RestorePath != nil {
		rriItemID2Resource, component2Resources := types.BuildClusterMap(resources, rris)
		rs, err = util.RandomResource(component2Resources["pd"])
		if err != nil {
			goto DestroyCluster
		}
		if _, err := workload.RestoreData(*wr.RestorePath, rs.IP, rriItemID2Resource[wr.RRIItemID].IP); err != nil {
			goto DestroyCluster
		}
	}
	dockerExecutor, containerID, out, err = workload.RunWorkload(cr, resources, rris, wr, artifactUUID, wr.Envs.Clone())
	if err != nil {
		zap.L().Error("run workload container failed",
			zap.ByteString("out", out.Bytes()),
			zap.Error(err))
		goto TearDown
	}
	defer func() {
		err := dockerExecutor.RmContainer(containerID)
		if err != nil {
			zap.L().Error("rm container failed", zap.String("container id", containerID), zap.Error(err))
		}
	}()
	if err = deploy.StopCluster(cr.Name); err != nil {
		zap.L().Error("stop cluster failed", zap.Error(err))
		goto TearDown
	}
TearDown:
	if err = m.archiveArtifacts(cr.ID, topo, wr, dockerExecutor, containerID, out, artifactUUID); err != nil {
		zap.L().Error("archive artifacts failed", zap.Error(err))
	}
DestroyCluster:
	if err = deploy.DestroyCluster(cr.Name); err != nil {
		zap.L().Error("destroy cluster failed", zap.Error(err))
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
	clusterRequestID, err := strconv.ParseUint(vars["cluster_id"], 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1000000))
	if err != nil {
		fail(w, err)
		return
	}
	r.Body.Close()
	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		fail(w, err)
		return
	}
	cr, err := m.Cluster.GetClusterRequest(m.Cluster.DB.DB, uint(clusterRequestID))
	if err != nil {
		fail(w, err)
		return
	}
	if cr.Status != types.ClusterRequestStatusRunning {
		fail(w, fmt.Errorf("cluster %d expect %s, but got %s", clusterRequestID, types.ClusterRequestStatusRunning, cr.Status))
		return
	}
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
	clusterRequestID := vars["cluster_id"]
	crID, err := strconv.ParseUint(clusterRequestID, 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	result, err := m.Cluster.FindWorkloadReportsByClusterRequestID(uint(crID))
	if err != nil {
		fail(w, err)
		return
	}
	okJSON(w, result)
}
