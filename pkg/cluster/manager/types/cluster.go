package types

import "github.com/jinzhu/gorm"

const (
	ClusterStatusReady       = "READY"
	ClusterTopoStatusReady   = "READY"
	ClusterTopoStatusOnline  = "ONLINE"
	ClusterTopoStatusOffline = "OFFLINE"
)

type ClusterRequest struct {
	gorm.Model
	Config    string `gorm:"column:config;type:text"`
	Version   string `gorm:"column:version;type:varchar(255);not null"`
	PDVersion string `gorm:"column:pd_version;type:varchar(255)"`
	RRID      uint   `gorm:"column:rr_id;not null"`
	Status    string `gorm:"column:status;type:varchar(255);not null"`
}

// ClusterRequestTopology defines which component is installed on a Resource.
type ClusterRequestTopology struct {
	gorm.Model
	Component  string `gorm:"column:component;type:varchar(255);not null"`
	DeployPath string `gorm:"column:deploy_path;type:varchar(255);not null"`
	CRID       uint   `gorm:"column:cr_id;not null"`
	RRIItemID  uint   `gorm:"column:rri_item_id;not null"`
	// READY
	// OFFLINE
	// ONLINE
	Status string `gorm:"column:status;not null"`
}
