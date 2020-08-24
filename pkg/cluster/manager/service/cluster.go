package service

import (
	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
	"sync"
)

type Cluster struct {
	DB   *mysql.DB
	lock sync.Mutex
}

func (c *Cluster) List() ([]types.ClusterRequest, error) {
	var result []types.ClusterRequest
	if err := c.DB.Find(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (c *Cluster) GetClusterRequestByResourceRequestID(rrID uint) (*types.ClusterRequest, error) {
	var result types.ClusterRequest
	if err := c.DB.First(&result, "rr_id = ?", rrID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (c *Cluster) GetClusterRequestTopoByClusterRequestID(crID uint) ([]types.ClusterRequestTopology, error) {
	var result []types.ClusterRequestTopology
	if err := c.DB.Find(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}
