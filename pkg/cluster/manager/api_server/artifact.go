package api_server

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/juju/errors"
	"github.com/rogpeppe/fastuuid"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/artifacts"
	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

func (m *Manager) getWorkloadArtifacts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterRequestID, err := strconv.ParseUint(vars["cluster_id"], 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	as, err := m.Artifacts.FindArtifacts(m.DB.DB, "cr_id = ?", clusterRequestID)
	if err != nil {
		fail(w, err)
		return
	}
	okJSON(w, as)
}

func (m *Manager) rebuildMonitoring(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterRequestID, err := strconv.ParseUint(vars["cluster_id"], 10, 64)
	if err != nil {
		fail(w, err)
		return
	}
	uuid := vars["uuid"]
	as, err := m.Artifacts.FindArtifacts(m.DB.DB, "cr_id = ? AND uuid = ?", clusterRequestID, uuid)
	if err != nil {
		fail(w, err)
		return
	}
	if len(as) != 1 {
		fail(w, errors.NotFoundf("monitor data of %d and %d not found", clusterRequestID, uuid))
		return
	}
	err = artifacts.RebuildMonitoringOnK8s(uuid)
	if err != nil {
		fail(w, err)
		return
	}
	ok(w, "success")
}

func (m *Manager) archiveArtifacts(crID uint, topos *deploy.Topology) error {
	artifactUUID := fastuuid.MustNewGenerator().Hex128()
	if err := artifacts.ArchiveMonitorData(artifactUUID, topos); err != nil {
		return errors.Trace(err)
	}
	if err := m.Artifacts.CreateArtifacts(m.DB.DB, &types.Artifacts{
		CRID: crID,
		UUID: artifactUUID,
	}); err != nil {
		return errors.Trace(err)
	}
	zap.L().Info("upload monitor data success", zap.String("uuid", artifactUUID))
	return nil
}
