package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/deploy"
	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/service"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"github.com/pingcap/tipocket/pkg/cluster/manager/workload"
)

type Manager struct {
	DB       *mysql.DB
	Resource *service.Resource
	Cluster  *service.Cluster
	sync.Mutex
}

func New(dsn string) (*Manager, error) {
	db, err := mysql.Open(dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Manager{
		DB: db,
		Resource: &service.Resource{
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
	m.DB.AutoMigrate(&types.Resource{}, &types.ResourceRequest{}, &types.ResourceRequestItem{},
		&types.ClusterRequest{}, &types.ClusterRequestTopology{},
		&types.WorkloadRequest{}, &types.WorkloadReport{},
	)
}

func (m *Manager) runServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		json.NewEncoder(writer).Encode(map[string]bool{"ok": true})
	})
	r.HandleFunc("/api/cluster/list", m.clusterList)
	r.HandleFunc("/api/cluster/resource/{name}", m.clusterResourceByName)
	r.HandleFunc("/api/cluster/deploy/{name}", m.clusterDeploy)
	r.HandleFunc("/api/cluster/destroy/{name}", m.clusterDestroy)
	r.HandleFunc("/api/cluster/scale_out/{name}/{id}/{component}", m.clusterScaleOut)
	r.HandleFunc("/api/cluster/workload/{name}", m.runWorkload)
	r.HandleFunc("/api/cluster/workload/{name}/result", m.uploadWorkloadResult).Methods("POST")
	r.HandleFunc("/api/cluster/workload/{name}/result", m.getWorkloadResult).Methods("GET")

	srv := &http.Server{
		Addr:         "127.0.0.1:8000",
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func (m *Manager) clusterList(w http.ResponseWriter, r *http.Request) {
	cluster, err := m.Cluster.ListClusterRequests()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cluster)
}

func (m *Manager) clusterResourceByName(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()
	vars := mux.Vars(r)
	name := vars["name"]
	rr, err := m.Resource.FindResourceRequestItemsByResourceRequestName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request item by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rr)
}

func (m *Manager) clusterDeploy(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]
	rr, err := m.Resource.FindResourceRequestByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by resource request id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	if cr.Status != types.ClusterStatusReady {
		http.Error(w, fmt.Sprintf("cluster %s expect %s, but got %s", rr.Name, types.ClusterStatusReady, cr.Status), http.StatusInternalServerError)
		return
	}
	rris, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request items by rr_id %d failed, err: %v", rr.ID, err.Error()), http.StatusInternalServerError)
		return
	}
	var rids []uint
	for _, rri := range rris {
		rids = append(rids, rri.RID)
	}
	rs, err := m.Resource.FindResourcesByIDs(rids)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resources by ids %v failed, err: %v", rids, err), http.StatusInternalServerError)
		return
	}
	crts, err := m.Cluster.FindClusterRequestToposByCRID(cr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request topology by cr_id %d failed, err: %v", cr.ID, err), http.StatusInternalServerError)
		return
	}
	if err := deploy.TryDeployCluster(rr.Name, rs, rris, cr, crts); err != nil {
		http.Error(w, fmt.Sprintf("try deploy cluster %s failed, err: %v", rr.Name, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := m.setOnline(rris, crts); err != nil {
		http.Error(w, fmt.Sprintf("set online failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "deploy cluster %s success", rr.Name)
}

func (m *Manager) clusterDestroy(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]
	rr, err := m.Resource.FindResourceRequestByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by resource request id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	if cr.Status != types.ClusterStatusReady {
		http.Error(w, fmt.Sprintf("cluster %s expect %s, but got %s", rr.Name, types.ClusterStatusReady, cr.Status), http.StatusInternalServerError)
		return
	}

	rris, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request items by rr_id %d failed, err: %v", rr.ID, err.Error()), http.StatusInternalServerError)
		return
	}
	crts, err := m.Cluster.FindClusterRequestToposByCRID(cr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request topology by cr_id %d failed, err: %v", cr.ID, err), http.StatusInternalServerError)
		return
	}

	if err := deploy.TryDestroyCluster(rr.Name); err != nil {
		http.Error(w, fmt.Sprintf("try destroy cluster %s failed, err: %v", rr.Name, err.Error()), http.StatusInternalServerError)
		return
	}
	if err := m.setOffline(rris, crts); err != nil {
		http.Error(w, fmt.Sprintf("set offline failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "destry cluster %s success", rr.Name)
}

func (m *Manager) clusterScaleOut(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	component := vars["component"]

	rr, err := m.Resource.FindResourceRequestByName(name)
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

func (m *Manager) setOffline(rris []*types.ResourceRequestItem, crts []*types.ClusterRequestTopology) error {
	for _, rri := range rris {
		rri.Components = ""
	}
	for _, crt := range crts {
		crt.Status = types.ClusterTopoStatusReady
	}
	return m.Resource.UpdateResourceRequestItemsAndClusterRequestTopos(rris, crts)
}

func (m *Manager) runWorkload(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]

	rr, err := m.Resource.FindResourceRequestByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by resource request id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	if cr.Status != types.ClusterStatusReady {
		http.Error(w, fmt.Sprintf("cluster %s expect %s, but got %s", rr.Name, types.ClusterStatusReady, cr.Status), http.StatusInternalServerError)
		return
	}
	rris, err := m.Resource.FindResourceRequestItemsByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request items by rr_id %d failed, err: %v", rr.ID, err.Error()), http.StatusInternalServerError)
		return
	}
	var rids []uint
	for _, rri := range rris {
		rids = append(rids, rri.RID)
	}
	rs, err := m.Resource.FindResourcesByIDs(rids)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resources by ids %v failed, err: %v", rids, err), http.StatusInternalServerError)
		return
	}
	wr, err := m.Cluster.GetClusterWorkloadByClusterRequestID(cr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("find workload by cr_id %d failed, err: %v", cr.ID, err), http.StatusInternalServerError)
		return
	}
	_, _, err = workload.TryRunWorkload(rr.Name, rs, rris, wr)
	if err != nil {
		http.Error(w, fmt.Sprintf("try run workload failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "run workload success")
}

func (m *Manager) uploadWorkloadResult(w http.ResponseWriter, r *http.Request) {
	type Result struct {
		Data      string  `json:"data"`
		PlainText *string `json:"plaintext"`
	}

	m.Lock()
	defer m.Unlock()

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
	rr, err := m.Resource.FindResourceRequestByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByRRID(rr.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get cluster request by resource request id %d failed, err: %v", rr.ID, err), http.StatusInternalServerError)
		return
	}
	//if cr.Status != types.ClusterStatusReady {
	//	http.Error(w, fmt.Sprintf("cluster %s expect %s, but got %s", rr.Name, types.ClusterStatusReady, cr.Status), http.StatusInternalServerError)
	//	return
	//}
	wr := &types.WorkloadReport{
		CRID:      cr.ID,
		Data:      result.Data,
		PlainText: result.PlainText,
	}
	if err := m.Cluster.AddWorkloadReport(wr); err != nil {
		http.Error(w, fmt.Sprintf("upload workload result %+v failed: %v", wr, err), http.StatusInternalServerError)
		return
	}
	ok(w, "upload workload result success")
}

func (m *Manager) getWorkloadResult(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	defer m.Unlock()

	vars := mux.Vars(r)
	name := vars["name"]

	rr, err := m.Resource.FindResourceRequestByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("find resource request by name %s failed, err: %v", name, err), http.StatusInternalServerError)
		return
	}
	cr, err := m.Cluster.GetClusterRequestByRRID(rr.ID)
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

func ok(w http.ResponseWriter, format string, a ...interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, format, a...)
}

func okJSON(w http.ResponseWriter, a interface{}) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a)
}
