package mysql

import (
	"github.com/jinzhu/gorm"
	"github.com/juju/errors"
)

// DB ...
type DB struct {
	*gorm.DB
}

// Open ...
func Open(dsn string) (*DB, error) {
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &DB{
		db,
	}, nil
}
