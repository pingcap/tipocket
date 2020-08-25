package mysql

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/juju/errors"
)

type DB struct {
	*gorm.DB
}

func Open(dsn string) (*DB, error) {
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &DB{
		db,
	}, nil
}
