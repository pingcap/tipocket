package types

import "github.com/jinzhu/gorm"

// Artifacts ...
// 1. prometheus/grafana archive data
// 2. TiDB/TiKV/PD and workloads logs
type Artifacts struct {
	gorm.Model
	CRID uint   `gorm:"column:cr_id" json:"cr_id"`
	UUID string `gorm:"column:uuid" json:"uuid"`
}
