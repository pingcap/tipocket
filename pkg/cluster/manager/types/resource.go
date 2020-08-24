package types

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/jinzhu/gorm"
)

type Spec struct {
	CPU  string
	Mem  string
	Disk string
}

func (s *Spec) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), &s)
}

func (s *Spec) Value() (driver.Value, error) {
	val, err := json.Marshal(s)
	return string(val), err
}

func (s *Spec) Suit(o *Spec) bool {
	// FIXME: @mahjonp
	return true
}

type Resource struct {
	gorm.Model
	IP    string `gorm:"column:ip;type:varchar(20);unique;not null"`
	Spec  Spec   `gorm:"column:spec;type:longtext;not null"`
	RRIID uint   `gorm:"column:rri_id"`
}

type ResourceRequest struct {
	gorm.Model
	Name   string `gorm:"column:name;type:varchar(255);unique;not null"`
	Status string `gorm:"column:status;type:varchar(255);not null"`
}

type ResourceRequestItem struct {
	gorm.Model
	ItemID uint   `gorm:"column:item_id;unique;not null"`
	Spec   Spec   `gorm:"column:spec;type:longtext;not null"`
	Status string `gorm:"column:status;type:varchar(255);not null"`
	RRID   uint   `gorm:"column:rr_id;not null"`
	RID    uint   `gorm:"column:r_id;not null"`
}
