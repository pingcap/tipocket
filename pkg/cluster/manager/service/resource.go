package service

import (
	"sync"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

type ResourceRequest struct {
	DB   *mysql.DB
	lock sync.Mutex
}

func (rr *ResourceRequest) FindByName(name string) (*types.ResourceRequest, error) {
	var result types.ResourceRequest
	if err := rr.DB.First(&result, "name = ?", name).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}
