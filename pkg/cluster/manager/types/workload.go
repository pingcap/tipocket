package types

import "github.com/jinzhu/gorm"

type WorkloadRequest struct {
	gorm.Model
	DockerImage string `gorm:"column:docker_image;type:varchar(255);not null"`
	Cmd         string `gorm:"column:cmd;type:varchar(255);not null"`
	Args        string `gorm:"column:args;type:varchar(1024);not null"`
	Status      string `gorm:"column:status;not null"`
	CRID        uint   `gorm:"column:cr_id;not null"`
	RRIItemID   uint   `gorm:"column:rri_item_id;not null"`
}

type WorkloadReport struct {
	gorm.Model
	CRID      uint    `gorm:"column:cr_id;not null" json:"cr_id"`
	Data      string  `gorm:"column:result;type:longtext;not null" json:"data"`
	PlainText *string `gorm:"column:plaintext" json:"plaintext,omitempty"`
}
