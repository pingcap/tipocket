package service

import (
	"sync"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

type Resource struct {
	DB   *mysql.DB
	lock sync.Mutex
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

func (rr *Resource) FindResourcesByIDs(ids []uint) ([]types.Resource, error) {
	var result []types.Resource
	if err := rr.DB.Where("id IN (?)", ids).Find(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}
