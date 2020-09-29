package types

import (
	"github.com/jinzhu/gorm"
)

const (
	// ResourceStatusReady ...
	ResourceStatusReady = "READY"
	// ResourceStatusBinding ...
	ResourceStatusBinding = "BINDING"
	// ResourceRequestStatusIdle ...
	ResourceRequestStatusIdle = "IDLE"
	// ResourceRequestStatusPending ...
	ResourceRequestStatusPending = "PENDING"
	// ResourceRequestStatusReady ...
	ResourceRequestStatusReady = "READY"
	// ResourceRequestStatusDisable ...
	ResourceRequestStatusDisable = "DISABLE"
)

// Resource means (physical- or vm-)machine
type Resource struct {
	gorm.Model
	IP           string `gorm:"column:ip;type:varchar(20);unique;not null" json:"ip"`
	Username     string `gorm:"column:username;type:varchar(50);not null" json:"username"`
	InstanceType string `gorm:"column:instance_type;type:varchar(100);not null" json:"instance_type"`
	Status       string `gorm:"column:status;type:varchar(255);not null" json:"status"`
	RRID         uint   `gorm:"column:rr_id" json:"rr_id"`
}

// ResourceRequest ...
type ResourceRequest struct {
	gorm.Model
	Name   string `gorm:"column:name;type:varchar(255);unique;not null"`
	Status string `gorm:"column:status;type:varchar(255);not null"`
	CRID   uint   `gorm:"column:cr_id"`
}

// ResourceRequestItem ...
type ResourceRequestItem struct {
	gorm.Model
	ItemID       uint   `gorm:"column:item_id;unique;not null" json:"item_id"`
	InstanceType string `gorm:"column:instance_type;type:varchar(100);not null" json:"instance_type"`
	RRID         uint   `gorm:"column:rr_id;not null" json:"rr_id"`
	RID          uint   `gorm:"column:r_id" json:"r_id"`
	Components   string `gorm:"column:components" json:"components"` // Components records which *_servers are serving on this machine
}

// ResourceRequestItemWithIP aggregates ResourceRequestItem with the ip field
type ResourceRequestItemWithIP struct {
	gorm.Model
	ItemID       uint   `gorm:"column:item_id;unique;not null" json:"item_id"`
	InstanceType string `gorm:"column:instance_type;type:varchar(100);not null" json:"instance_type"`
	RRID         uint   `gorm:"column:rr_id;not null" json:"rr_id"`
	RID          uint   `gorm:"column:r_id" json:"r_id"`
	Components   string `gorm:"column:components" json:"components"`
	IP           string `json:"ip"`
}
