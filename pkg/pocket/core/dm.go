// Copyright 2020 PingCAP, Inc.
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

package core

import (
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tipocket/pkg/pocket/util"
)

func (c *Core) dmTestSingleCompareData(delay bool) (bool, error) {
	c.execMutex.Lock()
	c.Lock()
	defer func() {
		c.Unlock()
		c.execMutex.Unlock()
	}()

	if delay {
		time.Sleep(5 * time.Second)
	}

	schema, err := c.coreExec.GetConn().FetchSchema(c.dbname)
	if err != nil {
		return false, err
	}
	sqls := makeCompareSQLs(schema)
	for _, sql := range sqls {
		err := wait.PollImmediate(1*time.Minute, 10*time.Minute, func() (done bool, err error) {
			if err = c.coreExec.DMSelectEqual(sql, c.coreExec.GetConn1(), c.coreExec.GetConn3()); err != nil {
				if errors.Cause(err) == util.ErrExactlyNotSame {
					log.Errorf("inconsistency when exec %s compare data %+v", sql, err)
				}
				log.Errorf("testing occurred an error and will retry later, %+v", err)
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			return false, errors.Trace(err)
		}
	}
	log.Info("consistency check pass")
	return true, nil
}
