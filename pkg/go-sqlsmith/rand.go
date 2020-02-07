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

import (
	"fmt"
	"time"
)

func (s *SQLSmith) rd(n int) int {
	return s.Rand.Intn(n)
}

func (s *SQLSmith) rdRange(n, m int) int {
	if m < n {
		n, m = m, n
	}
	return n + s.Rand.Intn(m-n)
}

func (s *SQLSmith) rdFloat64() float64 {
	return s.Rand.Float64()
}

func (s *SQLSmith) rdDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2100, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := s.Rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

func (s *SQLSmith) getSubTableName() string {
	name := fmt.Sprintf("ss_sub_%d", s.subTableIndex)
	s.subTableIndex++
	return name
}

func (s *SQLSmith) rdString(length int) string {
	res := ""
	for i := 0; i < length; i++ {
		charCode := s.rdRange(33, 127)
		// char '\' and '"' should be escaped
		if charCode == 92 || charCode == 34 {
			res = fmt.Sprintf("%s%s", res, "\\")
		}
		res = fmt.Sprintf("%s%s", res, string(rune(charCode)))
	}
	return res
}
