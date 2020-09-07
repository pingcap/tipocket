package types

import "github.com/jinzhu/gorm"

const (
	// ClusterRequestStatusReady ...
	ClusterRequestStatusReady = "READY"
	// ClusterRequestStatusRunning ...
	ClusterRequestStatusRunning = "RUNNING"
	// ClusterRequestStatusDone ...
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
