package service

import (
	"github.com/jinzhu/gorm"
	"sync"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

type Cluster struct {
	DB   *mysql.DB
	lock sync.Mutex
}

func (c *Cluster) ListClusterRequests() ([]types.ClusterRequest, error) {
	var result []types.ClusterRequest
	if err := c.DB.Find(&result).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (c *Cluster) GetLastClusterRequestByRRID(rrID uint) (*types.ClusterRequest, error) {
	var result types.ClusterRequest
	if err := c.DB.Last(&result, "rr_id = ?", rrID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (c *Cluster) CreateClusterRequest(tx *gorm.DB, cr *types.ClusterRequest) error {
	cr.Status = types.ClusterRequestStatusReady
	if err := tx.Create(cr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *Cluster) UpdateClusterRequest(tx *gorm.DB, cr *types.ClusterRequest) error {
	if err := tx.Save(cr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *Cluster) FindClusterRequestToposByCRID(crID uint) ([]*types.ClusterRequestTopology, error) {
	var result []*types.ClusterRequestTopology
	if err := c.DB.Find(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

func (c *Cluster) CreateClusterRequestTopos(tx *gorm.DB, crts []*types.ClusterRequestTopology) error {
	for _, crt := range crts {
		crt.Status = types.ClusterTopoStatusReady
		if err := tx.Create(crt).Error; err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *Cluster) GetClusterWorkloadByClusterRequestID(crID uint) (*types.WorkloadRequest, error) {
	var result types.WorkloadRequest
	if err := c.DB.First(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

func (c *Cluster) CreateWorkload(tx *gorm.DB, cw *types.WorkloadRequest) error {
	cw.Status = types.WorkloadStatusReady
	if err := tx.Create(cw).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *Cluster) CreateWorkloadReport(wr *types.WorkloadReport) error {
	if err := c.DB.Create(wr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *Cluster) FindWorkloadReportsByClusterRequestID(crID uint) ([]*types.WorkloadReport, error) {
	var result []*types.WorkloadReport
	if err := c.DB.Find(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}
