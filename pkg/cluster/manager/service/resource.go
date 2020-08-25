package service

import (
	"github.com/jinzhu/gorm"
	"sync"

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

func (rr *Resource) FindResourceRequestByName(name string) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := rr.DB.First(&result, "name = ?", name).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (rr *Resource) FindResourceRequestItemsByRRID(rrid uint) ([]types.ResourceRequestItem, error) {
	var result []types.ResourceRequestItem
	if err := rr.DB.Find(&result, "rr_id = ?", rrid).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (rr *Resource) FindResourceRequestItemsByResourceRequestName(name string) ([]types.ResourceRequestItem, error) {
	var result []types.ResourceRequestItem
	if err := rr.DB.Raw("SELECT * FROM resource_request_items WHERE rr_id = (SELECT id FROM resource_requests WHERE name = ?)", name).
		Scan(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (rr *Resource) UpdateResourceRequestItemsAndClusterRequestTopos(
	rris []types.ResourceRequestItem,
	crts []types.ClusterRequestTopology) error {
	return rr.DB.Transaction(func(tx *gorm.DB) error {
		for _, rri := range rris {
			if err := tx.Save(rri).Error; err != nil {
				return errors.Trace(err)
			}
		}
		for _, crt := range crts {
			if err := tx.Save(crt).Error; err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})
}
