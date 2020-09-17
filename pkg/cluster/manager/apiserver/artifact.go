package apiserver

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
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
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
	err = artifacts.RebuildMonitoringOnK8s(clusterRequestID, uuid)
	if err != nil {
		fail(w, err)
		return
	}
	ok(w, "success")
}

func (m *Manager) archiveArtifacts(
	crID uint,
	topos *deploy.Topology,
	wr *types.WorkloadRequest,
	dockerExecutor *util.DockerExecutor,
	containerID string) error {
	artifactUUID := fastuuid.MustNewGenerator().Hex128()
	s3Client, err := artifacts.NewS3Client()
	if err != nil {
		return errors.Trace(err)
	}
	if err := artifacts.ArchiveMonitorData(s3Client, crID, artifactUUID, topos); err != nil {
		return errors.Trace(err)
	}
	if wr.ArtifactDir != nil {
		err := artifacts.ArchiveWorkloadData(s3Client, dockerExecutor, containerID, crID, artifactUUID, *wr.ArtifactDir)
		if err != nil {
			return errors.Trace(err)
		}
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
