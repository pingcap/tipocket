package service

import (
	"sync"

	"github.com/jinzhu/gorm"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

type Resource struct {
	DB   *mysql.DB
	lock sync.Mutex
}

func (rr *Resource) FindResourcesByIDs(ids []uint) ([]types.Resource, error) {
	var result []types.Resource
	if err := rr.DB.Where("id IN (?)", ids).Find(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (rr *Resource) GetResourceByID(id uint) (*types.Resource, error) {
	var result types.Resource
	if err := rr.DB.First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (rr *Resource) GetResourceRequestByName(tx *gorm.DB, name string) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&result, "name = ?", name).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (rr *Resource) FindResourceRequestItemsByRRID(rrid uint) ([]*types.ResourceRequestItem, error) {
	var result []*types.ResourceRequestItem
	if err := rr.DB.Find(&result, "rr_id = ?", rrid).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (rr *Resource) GetResourceRequestItemByID(id uint) (*types.ResourceRequestItem, error) {
	var result types.ResourceRequestItem
	if err := rr.DB.First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (rr *Resource) FindResourceRequestItemsByResourceRequestName(name string) ([]*types.ResourceRequestItem, error) {
	var result []*types.ResourceRequestItem
	if err := rr.DB.Raw("SELECT * FROM resource_request_items WHERE rr_id = (SELECT id FROM resource_requests WHERE name = ?)", name).
		Scan(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

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
