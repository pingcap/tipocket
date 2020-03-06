package util

import (
	"github.com/go-sql-driver/mysql"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/parser/terror"
)

// IsErrDupEntry returns true if error code = 1062
func IsErrDupEntry(err error) bool {
	return isMySQLError(err, 1062)
}

// IsErrTableNotExists checks whether err is TableNotExists error
func IsErrTableNotExists(err error) bool {
	return isMySQLError(err, 1146)
}

func isMySQLError(err error, code uint16) bool {
	err = originError(err)
	e, ok := err.(*mysql.MySQLError)
	return ok && e.Number == code
}

// originError return original error
func originError(err error) error {
	for {
		e := errors.Cause(err)
		if e == err {
			break
		}
		err = e
	}
	return err
}

// IgnoreErrors returs true if ignoreErrs contains err
func IgnoreErrors(err error, ignoreErrs []terror.ErrCode) bool {
	mysqlErr, ok := originError(err).(*mysql.MySQLError)
	if !ok {
		log.Errorf("[error: %v] is not mysql error", err)
		return false
	}
	errCode := terror.ErrCode(mysqlErr.Number)
	for _, ignoreErrC := range ignoreErrs {
		if errCode == ignoreErrC {
			return true
		}
	}
	return false
}
