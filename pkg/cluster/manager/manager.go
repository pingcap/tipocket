package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/service"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

var Mrg Manager

type Manager struct {
	DB              *mysql.DB
	ResourceRequest *service.ResourceRequest
	Cluster         *service.Cluster
	sync.Mutex
}

func New(dsn string) (*Manager, error) {
	db, err := mysql.Open(dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Manager{
		DB: db,
		ResourceRequest: &service.ResourceRequest{
			DB: db,
		},
		Cluster: &service.Cluster{
			DB: db,
		},
	}, nil
}

func (m *Manager) Run() (err error) {
	defer m.DB.Close()
	m.migrate()
	m.runServer()
	return nil
}

func (m *Manager) migrate() {
	m.DB.AutoMigrate(&types.Resource{}, &types.ResourceRequest{}, &types.ResourceRequestItem{}, &types.ClusterRequest{}, &types.ClusterRequestTopology{})
}

func (m *Manager) runServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(map[string]bool{"ok": true})
	})
	r.HandleFunc("/api/cluster/list", m.clusterList)
	r.HandleFunc("/api/cluster/deploy/{name}", m.clusterDeploy)

	srv := &http.Server{
		Addr:         "127.0.0.1:8000",
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func (m *Manager) clusterList(w http.ResponseWriter, r *http.Request) {
	cluster, err := m.Cluster.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cluster)
}

func (m *Manager) clusterDeploy(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]
	rr, err := m.ResourceRequest.FindByName(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByResourceRequestID(rr.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cr.Status != types.ClusterStatusReady {
		http.Error(w, fmt.Sprintf("cluster %s expect %s, got %s", rr.Name, types.ClusterStatusReady, cr.Status), http.StatusInternalServerError)
		return
	}
	crt, err := m.Cluster.GetClusterRequestTopoByClusterRequestID(cr.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
