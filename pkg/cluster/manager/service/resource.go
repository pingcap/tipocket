package service

import (
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// Resource ...
type Resource struct {
	DB   *mysql.DB
	lock sync.Mutex
}

// FindResources ...
func (rr *Resource) FindResources(tx *gorm.DB, where ...interface{}) ([]*types.Resource, error) {
	var result []*types.Resource
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Find(&result, where...).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// UpdateResource ...
func (rr *Resource) UpdateResource(tx *gorm.DB, r *types.Resource) error {
	return errors.Trace(tx.Save(r).Error)
}

// GetResourceByID ...
func (rr *Resource) GetResourceByID(id uint) (*types.Resource, error) {
	var result types.Resource
	if err := rr.DB.First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// GetResourceRequestByName ...
func (rr *Resource) GetResourceRequestByName(tx *gorm.DB, name string) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&result, "name = ?", name).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// GetResourceRequest ...
func (rr *Resource) GetResourceRequest(tx *gorm.DB, id uint) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// FindResourceRequests ...
func (rr *Resource) FindResourceRequests(tx *gorm.DB, where ...interface{}) ([]*types.ResourceRequest, error) {
	var result []*types.ResourceRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Find(&result, where...).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// FindResourceRequest ...
func (rr *Resource) FindResourceRequest(tx *gorm.DB, id uint) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// UpdateResourceRequest ...
func (rr *Resource) UpdateResourceRequest(tx *gorm.DB, r *types.ResourceRequest) error {
	return errors.Trace(tx.Save(r).Error)
}

// FindResourceRequestItemsByRRID ...
func (rr *Resource) FindResourceRequestItemsByRRID(rrid uint) ([]*types.ResourceRequestItem, error) {
	var result []*types.ResourceRequestItem
	if err := rr.DB.Find(&result, "rr_id = ?", rrid).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// GetResourceRequestItemByID ...
func (rr *Resource) GetResourceRequestItemByID(id uint) (*types.ResourceRequestItem, error) {
	var result types.ResourceRequestItem
	if err := rr.DB.First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// FindResourceRequestItemsByClusterRequestID ...
func (rr *Resource) FindResourceRequestItemsByClusterRequestID(id uint) ([]*types.ResourceRequestItem, error) {
	var result []*types.ResourceRequestItem
	if err := rr.DB.Raw("SELECT * FROM resource_request_items WHERE rr_id = (SELECT rr_id FROM cluster_requests WHERE id = ?)", id).
		Scan(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// UpdateResourceRequestItems ...
func (rr *Resource) UpdateResourceRequestItems(tx *gorm.DB, rris []*types.ResourceRequestItem) error {
	for _, rri := range rris {
		if err := tx.Save(rri).Error; err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// UpdateResourceRequestItemsAndClusterRequestTopos ...
func (rr *Resource) UpdateResourceRequestItemsAndClusterRequestTopos(
	rris []*types.ResourceRequestItem,
	crts []*types.ClusterRequestTopology) error {
	return rr.DB.Transaction(func(tx *gorm.DB) error {
		for _, rri := range rris {
			if err := tx.Model(rri).Update("components", rri.Components).Error; err != nil {
				return errors.Trace(err)
			}
		}
		for _, crt := range crts {
			if err := tx.Model(crt).Update("status", crt.Status).Error; err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})
}
