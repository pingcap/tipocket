package types

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/juju/errors"
)

type Args []string

func (j Args) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	valueString, err := json.Marshal(j)
	return string(valueString), err
}

func (j *Args) Scan(value interface{}) error {
	if err := json.Unmarshal(value.([]byte), j); err != nil {
		return errors.Trace(err)
	}
	return nil
}

type WorkloadRequest struct {
	gorm.Model
	// data preparation
	//PrepareDockerImage *string `gorm:"column:prepare_docker_image;type:varchar(1024)" json:"prepare_docker_image"`
	//PrepareCmd         *string `gorm:"column:prepare_cmd;type:varchar(255)" json:"prepare_cmd"`
	//PrepareArgs        Args    `gorm:"column:prepare_args;type:varchar(1024)" json:"prepare_args"`
	RestorePath *string `gorm:"column:restore_path;type:varchar(1024)" json:"restore_path"`
	DockerImage string  `gorm:"column:docker_image;type:varchar(255);not null" json:"docker_image"`
	Cmd         *string `gorm:"column:cmd;type:varchar(255)" json:"cmd"`
	Args        Args    `gorm:"column:args;type:varchar(1024)" json:"args"`
	Status      string  `gorm:"column:status;not null" json:"status"`
	CRID        uint    `gorm:"column:cr_id;not null" json:"cr_id"`
	RRIItemID   uint    `gorm:"column:rri_item_id;not null" json:"rri_item_id"`
}

type WorkloadReport struct {
	gorm.Model
	CRID      uint    `gorm:"column:cr_id;not null" json:"cr_id"`
	Data      string  `gorm:"column:result;type:longtext;not null" json:"data"`
	PlainText *string `gorm:"column:plaintext;type:longtext" json:"plaintext,omitempty"`
}
