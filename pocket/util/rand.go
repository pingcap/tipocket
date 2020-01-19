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

package util

import (
	"math/rand"
	"time"
)

var (
	seed = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// Rd implement Intn
func Rd(n int) int {
	return seed.Intn(n)
}

// RdRange rand between [min(n, m), max(n, m))
func RdRange(n, m int) int {
	if n == m {
		return n
	}
	if m < n {
		n, m = m, n
	}
	return n + seed.Intn(m-n)
}
