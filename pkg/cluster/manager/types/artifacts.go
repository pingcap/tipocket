package types

import "github.com/jinzhu/gorm"

// 1. prometheus/grafana archive data
// 2. TiDB/TiKV/PD and workloads logs
type Artifacts struct {
	gorm.Model
	RRID uint   `gorm:"column:rr_id" json:"rr_id"`
	UUID string `gorm:"column:uuid" json:"uuid"`
}
