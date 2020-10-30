package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
)

const (
	// ResourceStatusOffline ...
	ResourceStatusOffline = "OFFLINE"
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

// ResourceRequestQueue records the request queue of each resource
type ResourceRequestQueue struct {
	ResourceID         uint   `gorm:"column:r_id;unique;not null" json:"r_id"`
	PendingRequestList string `gorm:"column:pending_request_list;type:text" json:"pending_request_list"`
}

// RequestIDList alias []uint
type RequestIDList []uint

// Exist ...
func (r RequestIDList) Exist(id uint) bool {
	for _, n := range r {
		if n == id {
			return true
		}
	}
	return false
}

// IsEmpty ...
func (r RequestIDList) IsEmpty() bool {
	return len(r) == 0
}

// IsFirst ...
func (r RequestIDList) IsFirst(id uint) bool {
	if len(r) == 0 {
		return false
	}
	return r[0] == id
}

// PendingRequestIDList gets pending resource request id list
func (r *ResourceRequestQueue) PendingRequestIDList() (list RequestIDList) {
	if len(r.PendingRequestList) == 0 {
		return make(RequestIDList, 0)
	}
	for _, id := range strings.Split(r.PendingRequestList, ",") {
		n, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			panic(err)
		}
		list = append(list, uint(n))
	}
	return list
}

// PushPendingRequestID ...
func (r *ResourceRequestQueue) PushPendingRequestID(id uint) {
	if len(r.PendingRequestList) == 0 {
		r.PendingRequestList = fmt.Sprintf("%d", id)
	} else {
		r.PendingRequestList = fmt.Sprintf("%s,%d", r.PendingRequestList, id)
	}
}

// PopPendingRequestID ...
func (r *ResourceRequestQueue) PopPendingRequestID() uint {
	if len(r.PendingRequestList) == 0 {
		panic("a fatal bug exists")
	}
	id := r.PendingRequestIDList()[0]
	list := r.PendingRequestIDList()[1:]
	var pendingList string
	for idx, item := range list {
		if idx == 0 {
			pendingList = fmt.Sprintf("%d", item)
		} else {
			pendingList = fmt.Sprintf("%s,%d", pendingList, item)
		}
	}
	r.PendingRequestList = pendingList
	return id
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
	ItemID       uint   `gorm:"column:item_id;not null" json:"item_id"`
	InstanceType string `gorm:"column:instance_type;type:varchar(100);not null" json:"instance_type"`
	RRID         uint   `gorm:"column:rr_id;not null" json:"rr_id"`
	RID          uint   `gorm:"column:r_id" json:"r_id"`
	Static       bool   `gorm:"column:static" json:"static"`
	Components   string `gorm:"column:components" json:"components"` // Components records which *_servers are serving on this machine
}

// ResourceRequestItemWithIP aggregates ResourceRequestItem with the ip field
type ResourceRequestItemWithIP struct {
	gorm.Model
	ItemID       uint   `gorm:"column:item_id;not null" json:"item_id"`
	InstanceType string `gorm:"column:instance_type;type:varchar(100);not null" json:"instance_type"`
	RRID         uint   `gorm:"column:rr_id;not null" json:"rr_id"`
	RID          uint   `gorm:"column:r_id" json:"r_id"`
	Components   string `gorm:"column:components" json:"components"`
	IP           string `json:"ip"`
}
