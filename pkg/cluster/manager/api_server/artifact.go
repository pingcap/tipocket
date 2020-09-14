package api_server

import (
	"net/http"

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
	name := vars["name"]
	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	if err != nil {
		fail(w, err)
		return
	}
	as, err := m.Artifacts.FindArtifacts(m.DB.DB, "rr_id = ?", rr.ID)
	if err != nil {
		fail(w, err)
		return
	}
	okJSON(w, as)
}

func (m *Manager) rebuildMonitoring(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	uuid := vars["uuid"]
	rr, err := m.Resource.GetResourceRequestByName(m.DB.DB, name)
	as, err := m.Artifacts.FindArtifacts(m.DB.DB, "rr_id = ? AND uuid = ?", rr.ID, uuid)
	if err != nil {
		fail(w, err)
		return
	}
	if len(as) != 1 {
		fail(w, errors.NotFoundf("monitor data of %s and %s not found", name, uuid))
		return
	}
	err = artifacts.RebuildMonitoringOnK8s(uuid)
	if err != nil {
		fail(w, err)
		return
	}
	ok(w, "success")
}

func (m *Manager) archiveArtifacts(rrID uint, topos *deploy.Topology) error {
	artifactUUID := fastuuid.MustNewGenerator().Hex128()
	if err := artifacts.ArchiveMonitorData(artifactUUID, topos); err != nil {
		return errors.Trace(err)
	}
	if err := m.Artifacts.CreateArtifacts(m.DB.DB, &types.Artifacts{
		RRID: rrID,
		UUID: artifactUUID,
	}); err != nil {
		return errors.Trace(err)
	}
	zap.L().Info("upload monitor data success", zap.String("uuid", artifactUUID))
	return nil
}
