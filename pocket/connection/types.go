// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package connection

import (
	"database/sql"

	"github.com/juju/errors"
)

// QueryItem define query result
type QueryItem struct {
	Null      bool
	ValType   *sql.ColumnType
	ValString string
}

// MustSame compare tow QueryItem and return error if not same
func (q *QueryItem) MustSame(q1 *QueryItem) error {
	if (q == nil) != (q1 == nil) {
		return errors.Errorf("one is nil but another is not, self: %t, another: %t", q == nil, q1 == nil)
	}

	if q.Null != q1.Null {
		return errors.Errorf("one is NULL but another is not, self: %t, another: %t", q.Null, q1.Null)
	}

	if q.ValType.Name() != q1.ValType.Name() {
		return errors.Errorf("column type diff, self: %s, another: %s", q.ValType.Name(), q1.ValType.Name())
	}

	if q.ValString != q1.ValString {
		return errors.Errorf("column data diff, self: %s, another: %s", q.ValString, q1.ValString)
	}

	return nil
}
