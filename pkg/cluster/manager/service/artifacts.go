package service

import (
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/cluster/manager/mysql"
	"github.com/pingcap/tipocket/pkg/cluster/manager/types"
)

// Artifacts ...
type Artifacts struct {
	DB   *mysql.DB
	lock sync.Mutex
}

// CreateArtifacts creates an artifacts record
func (ats *Artifacts) CreateArtifacts(tx *gorm.DB, a *types.Artifacts) error {
	return errors.Trace(tx.Save(a).Error)
}

// FindArtifacts finds artifacts records
func (ats *Artifacts) FindArtifacts(tx *gorm.DB, where ...interface{}) ([]*types.Artifacts, error) {
	var result []*types.Artifacts
	if err := tx.Find(&result, where...).Error; err != nil {
		return nil, err
	}
	return result, nil
}
