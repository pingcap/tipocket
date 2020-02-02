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

package sqlsmith

import "errors"

// DataGenerator defines data generator
type DataGenerator struct {
	total    int
	batch    int
	curr     int
	sqlsmith *SQLSmith
}

// GenData returns data generator
func (s *SQLSmith) GenData(total, batch int) (*DataGenerator, error) {
	if s.currDB == "" {
		return nil, errors.New("no selected database")
	}
	_, ok := s.Databases[s.currDB]
	if !ok {
		return nil, errors.New("selected database's schema not loaded")
	}
	return &DataGenerator{
		total,
		batch,
		0,
		s,
	}, nil
}

// Next returns data batch
func (d *DataGenerator) Next() []string {
	if d.curr >= d.total {
		return []string{}
	} else if d.curr < d.total-d.batch {
		sqls, _ := d.sqlsmith.BatchData(d.batch, d.batch)
		d.curr += d.batch
		return sqls
	}
	sqls, _ := d.sqlsmith.BatchData(d.total-d.curr, d.batch)
	d.curr += (d.total - d.curr)
	return sqls
}
