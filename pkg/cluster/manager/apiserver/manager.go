package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/service"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/util"
)

// Manager ...
type Manager struct {
	DB        *mysql.DB
	Resource  *service.Resource
	Cluster   *service.Cluster
	Artifacts *service.Artifacts
	sync.Mutex
}

// New creates a manager instance
func New(dsn string) (*Manager, error) {
	db, err := mysql.Open(dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Manager{
		DB:        db,
		Resource:  &service.Resource{DB: db},
		Cluster:   &service.Cluster{DB: db},
		Artifacts: &service.Artifacts{DB: db},
	}, nil
}

// Run ...
func (m *Manager) Run() (err error) {
	defer m.DB.Close()
	m.migrate()
	m.runWatcher(context.TODO())
	m.runServer()
	return nil
}

func (m *Manager) migrate() {
	m.DB.AutoMigrate(&types.Resource{}, &types.ResourceRequest{}, &types.ResourceRequestItem{},
		&types.ClusterRequest{}, &types.ClusterRequestTopology{},
		&types.WorkloadRequest{}, &types.WorkloadReport{},
		&types.Artifacts{},
	)
}

func (m *Manager) runServer() {
	r := mux.NewRouter()
	r.HandleFunc("/api/cluster/list", m.clusterList)
	r.HandleFunc("/api/cluster/resource/{cluster_id}", m.clusterResourceByName)
	r.HandleFunc("/api/cluster/{name}", m.submitClusterRequest).Methods("POST")
	r.HandleFunc("/api/cluster/{cluster_id}", m.queryClusterRequest).Methods("GET")
	r.HandleFunc("/api/cluster/scale_out/{cluster_id}/{id}/{component}", m.clusterScaleOut)
	r.HandleFunc("/api/cluster/scale_in/{cluster_id}/{id}/{component}", m.clusterScaleIn)
	r.HandleFunc("/api/cluster/clean/{cluster_id}", m.clusterRebuild)
	r.HandleFunc("/api/cluster/workload/{cluster_id}/result", m.uploadWorkloadResult).Methods("POST")
	r.HandleFunc("/api/cluster/workload/{cluster_id}/result", m.getWorkloadResult).Methods("GET")
	r.HandleFunc("/api/cluster/workload/{cluster_id}/artifacts", m.getWorkloadArtifacts).Methods("GET")
	r.HandleFunc("/api/cluster/workload/{cluster_id}/artifacts/monitor/{uuid}", m.rebuildMonitoring).Methods("POST")

	srv := &http.Server{
		Addr:         util.Addr,
		Handler:      r,
		WriteTimeout: 15 * time.Minute,
		ReadTimeout:  15 * time.Minute,
	}
	log.Fatal(srv.ListenAndServe())
}

// runWatcher registers all watcher here
func (m *Manager) runWatcher(ctx context.Context) {
	for _, watcher := range []func(context.Context){
		m.PollPendingClusterRequests,
		m.PollPendingResourceRequests,
		m.PollReadyClusterRequests,
		m.PollPendingRebuildClusterRequests,
	} {
		go func(watcher func(ctx2 context.Context)) {
			watcher(ctx)
		}(watcher)
	}
}

func ok(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, msg)
}

func okJSON(w http.ResponseWriter, a interface{}) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a)
}

func fail(w http.ResponseWriter, err error) {
	zap.L().Debug("request failed", zap.Error(err))
	http.Error(w, errors.ErrorStack(err), http.StatusInternalServerError)
}
