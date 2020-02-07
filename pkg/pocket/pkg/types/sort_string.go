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

package types

// BySQL implement sort interface for string
type BySQL []string

func (a BySQL) Len() int      { return len(a) }
func (a BySQL) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySQL) Less(i, j int) bool {
	var (
		bi = []byte(a[i])
		bj = []byte(a[j])
	)

	for i := 0; i < min(len(bi), len(bj)); i++ {
		if bi[i] != bj[i] {
			return bi[i] < bj[i]
		}
	}
	return len(bi) < len(bj)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
