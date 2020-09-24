package types

import (
	"strings"

	"github.com/jinzhu/gorm"
)

const (
	// ClusterRequestStatusPending means that the request is in queue
	ClusterRequestStatusPending = "PENDING"
	// ClusterRequestStatusReady means that the request is met(get expected resources) but haven't run workloads
	ClusterRequestStatusReady = "READY"
	// ClusterRequestStatusRunning means that the request is running some workloads now
	ClusterRequestStatusRunning = "RUNNING"
	// ClusterRequestStatusPendingRebuild ...
	ClusterRequestStatusPendingRebuild = "PENDING-REBUILD"
	// ClusterRequestStatusRebuilding ...
	ClusterRequestStatusRebuilding = "REBUILDING"
	// ClusterRequestStatusDone means that the request is finished and resources
	ClusterRequestStatusDone = "DONE"

	// ClusterTopoStatusReady ...
	ClusterTopoStatusReady = "READY"
	// ClusterTopoStatusOnline ...
	ClusterTopoStatusOnline = "ONLINE"

	// WorkloadStatusReady ...
	WorkloadStatusReady = "READY"
	// WorkloadStatusRunning ...
	WorkloadStatusRunning = "RUNNING"
	// WorkloadStatusDone ...
	WorkloadStatusDone = "DONE"
)

// ClusterRequest ...
type ClusterRequest struct {
	gorm.Model
	Name        string `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Config      string `gorm:"column:config;type:text" json:"config"`
	Version     string `gorm:"column:version;type:varchar(255);not null" json:"version"`
	PDVersion   string `gorm:"column:pd_version;type:varchar(255)" json:"pd_version"`
	TiDBVersion string `gorm:"column:tidb_version;type:varchar(255)" json:"tidb_version"`
	TiKVVersion string `gorm:"column:tikv_version;type:varchar(255)" json:"tikv_version"`
	RRID        uint   `gorm:"column:rr_id;not null" json:"rr_id"`
	Status      string `gorm:"column:status;type:varchar(255);not null" json:"status"`
}

// Baseline ...
func (cr *ClusterRequest) Baseline() *ClusterRequest {
	return &ClusterRequest{
		Model: gorm.Model{
			ID:        cr.ID,
			CreatedAt: cr.CreatedAt,
			UpdatedAt: cr.UpdatedAt,
			DeletedAt: cr.DeletedAt,
		},
		Name:    cr.Name,
		Config:  cr.Config,
		Version: cr.Version,
		RRID:    cr.RRID,
		Status:  cr.Status,
	}
}

// ClusterRequestTopology defines which component is installed on a Resource.
type ClusterRequestTopology struct {
	gorm.Model
	Component  string `gorm:"column:component;type:varchar(255);not null" json:"component"`
	DeployPath string `gorm:"column:deploy_path;type:varchar(255);not null" json:"deploy_path"`
	CRID       uint   `gorm:"column:cr_id;not null" json:"cr_id"`
	RRIItemID  uint   `gorm:"column:rri_item_id;not null" json:"rri_item_id"`
	// READY
	// ONLINE
	Status string `gorm:"column:status;not null" json:"status"`
}

func BuildClusterMap(resources []*Resource, rris []*ResourceRequestItem) (
	rriItemID2Resource map[uint]*Resource, component2Resources map[string][]*Resource,
) {
	id2Resource := make(map[uint]*Resource)
	rriItemID2Resource = make(map[uint]*Resource)
	component2Resources = make(map[string][]*Resource)
	// resource request item id -> resource
	for idx, re := range resources {
		id2Resource[re.ID] = resources[idx]
	}
	for _, rri := range rris {
		rriItemID2Resource[rri.ItemID] = id2Resource[rri.RID]
		for _, component := range strings.Split(rri.Components, "|") {
			if _, ok := component2Resources[component]; !ok {
				component2Resources[component] = make([]*Resource, 0)
			}
			component2Resources[component] = append(component2Resources[component], id2Resource[rri.RID])
		}
	}
	return rriItemID2Resource, component2Resources
}
