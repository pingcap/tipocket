package service

import (
	"sync"

	"github.com/jinzhu/gorm"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// Cluster ...
type Cluster struct {
	DB   *mysql.DB
	lock sync.Mutex
}

// FindClusterRequests ...
func (c *Cluster) FindClusterRequests(tx *gorm.DB, where ...interface{}) ([]*types.ClusterRequest, error) {
	var result []*types.ClusterRequest
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Find(&result, where...).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// GetClusterRequest ...
func (c *Cluster) GetClusterRequest(tx *gorm.DB, id uint) (*types.ClusterRequest, error) {
	var result types.ClusterRequest
	if err := tx.Set("gorml:query_option", "FOR UPDATE").First(&result, "id = ?", id).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// GetLastClusterRequestByRRID ...
// Deprecated @FIXME(mahjonp) remove this method
func (c *Cluster) GetLastClusterRequestByRRID(rrID uint) (*types.ClusterRequest, error) {
	var result types.ClusterRequest
	if err := c.DB.Last(&result, "rr_id = ?", rrID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// CreateClusterRequest ...
func (c *Cluster) CreateClusterRequest(tx *gorm.DB, cr *types.ClusterRequest) error {
	cr.Status = types.ClusterRequestStatusReady
	if err := tx.Create(cr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

// UpdateClusterRequest ...
func (c *Cluster) UpdateClusterRequest(tx *gorm.DB, cr *types.ClusterRequest) error {
	if err := tx.Save(cr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

// FindClusterRequestToposByCRID ...
func (c *Cluster) FindClusterRequestToposByCRID(crID uint) ([]*types.ClusterRequestTopology, error) {
	var result []*types.ClusterRequestTopology
	if err := c.DB.Find(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}

// CreateClusterRequestTopos ...
func (c *Cluster) CreateClusterRequestTopos(tx *gorm.DB, crts []*types.ClusterRequestTopology) error {
	for _, crt := range crts {
		crt.Status = types.ClusterTopoStatusReady
		if err := tx.Create(crt).Error; err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// GetClusterWorkloadByClusterRequestID ...
func (c *Cluster) GetClusterWorkloadByClusterRequestID(crID uint) (*types.WorkloadRequest, error) {
	var result types.WorkloadRequest
	if err := c.DB.First(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return &result, nil
}

// CreateWorkloadRequest ...
func (c *Cluster) CreateWorkloadRequest(tx *gorm.DB, cw *types.WorkloadRequest) error {
	cw.Status = types.WorkloadStatusReady
	if err := tx.Create(cw).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

// UpdateWorkloadRequest ...
func (c *Cluster) UpdateWorkloadRequest(tx *gorm.DB, wr *types.WorkloadRequest) error {
	if err := tx.Save(wr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

// CreateWorkloadReport ...
func (c *Cluster) CreateWorkloadReport(wr *types.WorkloadReport) error {
	if err := c.DB.Create(wr).Error; err != nil {
		return errors.Trace(err)
	}
	return nil
}

// FindWorkloadReportsByClusterRequestID ...
func (c *Cluster) FindWorkloadReportsByClusterRequestID(crID uint) ([]*types.WorkloadReport, error) {
	var result []*types.WorkloadReport
	if err := c.DB.Find(&result, "cr_id = ?", crID).Error; err != nil {
		return nil, errors.Trace(err)
	}
	return result, nil
}
